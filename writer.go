package s3

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	MaxObjectSize = 5 * 1024 * 1024 * 1024 * 1024
	MinPartSize   = 5 * 1024 * 1024
	MaxPartSize   = 1<<31 - 1
	MaxNumParts   = 10000
)

const (
	nConcurrentUploads = 5
	nRetries           = 2
)

type Writer interface {
	io.WriteCloser

	// Abort aborts the current write/upload operation
	Abort() error
}

type writer struct {
	m        sync.Mutex
	once     sync.Once
	wg       sync.WaitGroup
	o        *object
	buf      *bytes.Buffer
	pc       chan *part
	partNum  int
	prepared bool
	closed   bool
	aborted  bool
	uploadId string
	err      error
	errAbort error
	xml      struct {
		XMLName string `xml:"CompleteMultipartUpload"`
		Part    []*part
	}
}

type part struct {
	buf []byte

	// xml
	PartNumber int
	ETag       string
}

func newWriter(o *object) *writer {
	return &writer{
		o:   o,
		buf: new(bytes.Buffer),
		pc:  make(chan *part, nConcurrentUploads),
	}
}

// prepare creates a multipart upload
func (w *writer) prepare() error {
	if w.prepared {
		return nil
	}
	req, err := http.NewRequest("POST", w.o.url("?uploads"), nil)
	if err != nil {
		return err
	}

	// detect mime type
	ext := filepath.Ext(w.o.key)
	contentType := "application/octet-stream"
	if v, ok := mimeTypes[ext]; ok {
		contentType = v
	}
	req.Header.Set(`Content-Type`, contentType)

	// sign and send
	w.o.s3.signRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c := resp.StatusCode; c != 200 {
		return newS3Error(resp, "could not create multipart upload: %d", c)
	}

	var result struct {
		UploadId string
	}
	err = xml.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}

	w.uploadId = result.UploadId
	w.prepared = true

	return nil
}

func (w *writer) Write(p []byte) (n int, err error) {
	w.m.Lock()
	defer w.m.Unlock()

	// prepare
	if !w.prepared {
		err := w.prepare()
		if err != nil {
			return 0, err
		}
	}

	// schedule worker once
	w.once.Do(func() {
		go w.schedule()
	})

	n, err = w.buf.Write(p)
	if err != nil {
		return
	}
	if w.buf.Len() > MinPartSize {
		w.flush()
	}
	return
}

func (w *writer) schedule() {
	for p := range w.pc {
		go w.uploadPartRetry(p)
	}
}

func (w *writer) flush() {
	b := w.buf.Bytes()
	if len(b) == 0 {
		return
	}
	w.buf = new(bytes.Buffer)
	w.partNum++
	p := &part{
		PartNumber: w.partNum,
		buf:        b,
	}
	w.xml.Part = append(w.xml.Part, p)
	w.wg.Add(1)
	w.pc <- p
}

func (w *writer) uploadPartRetry(p *part) {
	defer w.wg.Done()
	var err error
	for i := 0; i < nRetries; i++ {
		err = w.uploadPart(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		w.close(true)
	}
}

func (w *writer) uploadPart(p *part) error {
	buf := bytes.NewBuffer(p.buf)

	var uv = make(url.Values)
	uv.Set("partNumber", strconv.Itoa(p.PartNumber))
	uv.Set("uploadId", w.uploadId)

	url := w.o.url(`?` + uv.Encode())
	req, err := http.NewRequest("PUT", url, buf)
	if err != nil {
		return err
	}
	req.ContentLength = int64(buf.Len())

	w.o.s3.signRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c := resp.StatusCode; c != 200 {
		return newS3Error(resp, "could not upload part: %d", c)
	}

	// trim outer space and quotes from etag
	p.ETag = strings.Trim(resp.Header.Get("etag"), ` "`)

	return nil
}

func (w *writer) close(abort bool) error {
	w.m.Lock()
	defer w.m.Unlock()

	if w.closed {
		return nil
	}

	w.aborted = abort
	w.flush()
	w.wg.Wait()
	close(w.pc)
	w.closed = true

	if abort {
		return w.abort()
	}
	return w.complete()
}

func (w *writer) abort() error {
	uv := make(url.Values)
	uv.Set("uploadId", w.uploadId)
	url := w.o.url("?" + uv.Encode())

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	w.o.s3.signRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c := resp.StatusCode; c != 204 {
		return newS3Error(resp, "could not abort upload: %d", c)
	}

	return nil
}

func (w *writer) complete() error {

	b, err := xml.Marshal(w.xml)
	if err != nil {
		return err
	}

	uv := make(url.Values)
	uv.Set("uploadId", w.uploadId)

	url := w.o.url(`?` + uv.Encode())
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	w.o.s3.signRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c := resp.StatusCode; c != 200 {
		return newS3Error(resp, "could not complete upload: %d", c)
	}
	return nil
}

func (w *writer) Close() error {
	return w.close(false)
}

func (w *writer) Abort() error {
	return w.close(true)
}

type s3err struct {
	code    int
	text    string
	xmlBody string
}

func newS3Error(resp *http.Response, strFmt string, args ...interface{}) *s3err {
	var b bytes.Buffer
	if resp != nil {
		b.ReadFrom(resp.Body)
	}
	return &s3err{
		code:    resp.StatusCode,
		text:    fmt.Sprintf(strFmt, args...),
		xmlBody: b.String(),
	}
}

func (e *s3err) Error() string {
	return e.text
}
