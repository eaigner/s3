package s3

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// defined by amazon
const (
	minPartSize = 5 * 1024 * 1024
	maxPartSize = 1<<31 - 1 // for 32-bit use; amz max is 5GiB
	maxObjSize  = 5 * 1024 * 1024 * 1024 * 1024
	maxNPart    = 10000
)

const (
	concurrency = 5
	nTry        = 2
)

type part struct {
	r   io.ReadSeeker
	len int64

	// read by xml encoder
	PartNumber int
	ETag       string
}

type uploader struct {
	c        *S3
	path     string
	url      string
	UploadId string // written by xml decoder
	bufsz    int64
	buf      []byte
	off      int
	ch       chan *part
	part     int
	closed   bool
	aborted  bool
	err      error
	wg       sync.WaitGroup
	xml      struct {
		XMLName string `xml:"CompleteMultipartUpload"`
		Part    []*part
	}
}

// http://docs.amazonwebservices.com/AmazonS3/latest/dev/mpuoverview.html.
func newUploader(c *S3, path string) (u *uploader, err error) {
	u = &uploader{
		c:    c,
		path: path,
		url:  c.url(path),
	}

	u.bufsz = minPartSize
	r, err := http.NewRequest("POST", u.url+"?uploads", nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	u.c.signRequest(r)
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, newS3Error(resp)
	}
	err = xml.NewDecoder(resp.Body).Decode(u)
	if err != nil {
		return nil, err
	}
	u.ch = make(chan *part)
	for i := 0; i < concurrency; i++ {
		go u.worker()
	}
	return u, nil
}

func (u *uploader) Write(p []byte) (n int, err error) {
	if u.closed {
		return 0, syscall.EINVAL
	}
	if u.err != nil {
		return 0, u.err
	}
	for n < len(p) {
		if cap(u.buf) == 0 {
			u.buf = make([]byte, int(u.bufsz))
			// Increase part size (1.001x).
			// This lets us reach the max object size (5TiB) while
			// still doing minimal buffering for small objects.
			u.bufsz = min(u.bufsz+u.bufsz/1000, maxPartSize)
		}
		r := copy(u.buf[u.off:], p[n:])
		u.off += r
		n += r
		if u.off == len(u.buf) {
			u.flush()
		}
	}
	return n, nil
}

func (u *uploader) flush() {
	u.wg.Add(1)
	u.part++
	p := &part{bytes.NewReader(u.buf[:u.off]), int64(u.off), u.part, ""}
	u.xml.Part = append(u.xml.Part, p)
	u.ch <- p
	u.buf, u.off = nil, 0
}

func (u *uploader) worker() {
	for p := range u.ch {
		u.retryUploadPart(p)
	}
}

// Calls putPart up to nTry times to recover from transient errors.
func (u *uploader) retryUploadPart(p *part) {
	defer u.wg.Done()
	var err error
	for i := 0; i < nTry; i++ {
		p.r.Seek(0, 0)
		err = u.putPart(p)
		if err == nil {
			return
		}
	}
	u.err = err
}

// Uploads part p, reading its contents from p.r.
// Stores the ETag in p.ETag.
func (u *uploader) putPart(p *part) error {
	v := url.Values{}
	v.Set("partNumber", strconv.Itoa(p.PartNumber))
	v.Set("uploadId", u.UploadId)
	req, err := http.NewRequest("PUT", u.url+"?"+v.Encode(), p.r)
	if err != nil {
		return err
	}
	req.ContentLength = p.len
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	u.c.signRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newS3Error(resp)
	}
	s := resp.Header.Get("etag") // includes quote chars for some reason
	p.ETag = s[1 : len(s)-1]
	return nil
}

func (u *uploader) Close() error {
	if u.closed {
		return syscall.EINVAL
	}
	if cap(u.buf) > 0 {
		u.flush()
	}
	u.wg.Wait()
	close(u.ch)
	u.closed = true
	if u.aborted || u.err != nil {
		u.abort()
		return u.err
	}

	body, err := xml.Marshal(u.xml)
	if err != nil {
		return err
	}
	b := bytes.NewBuffer(body)
	v := url.Values{}
	v.Set("uploadId", u.UploadId)
	req, err := http.NewRequest("POST", u.url+"?"+v.Encode(), b)
	if err != nil {
		return err
	}
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	u.c.signRequest(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newS3Error(resp)
	}
	resp.Body.Close()
	return nil
}

func (u *uploader) Abort() error {
	u.aborted = true
	return u.Close()
}

func (u *uploader) abort() {
	// TODO(kr): devise a reasonable way to report an error here in addition
	// to the error that caused the abort.
	v := url.Values{}
	v.Set("uploadId", u.UploadId)
	s := u.url + "?" + v.Encode()
	req, err := http.NewRequest("DELETE", s, nil)
	if err != nil {
		return
	}
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	u.c.signRequest(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
