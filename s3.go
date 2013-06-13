package s3

import (
	"io"
	"net/http"
	"time"
)

type Object struct {
	io.WriteCloser
	c    *Conf
	Path string
}

// Delete deletes the S3 object
func (o *Object) Delete() error {
	_, err := o.request("DELETE", 204)
	return err
}

// Writer returns a new WriteCloser you can write to.
// The written data will be uploaded as a multipart request.
func (o *Object) Writer() (io.WriteCloser, error) {
	return newUploader(o.c, o.Path)
}

// Reader returns a new ReadCloser you can read from.
func (o *Object) Reader() (io.ReadCloser, http.Header, error) {
	resp, err := o.request("GET", 200)
	if err != nil {
		return nil, nil, err
	}
	return resp.Body, resp.Header, nil
}

func (o *Object) request(method string, expectCode int) (*http.Response, error) {
	req, err := http.NewRequest(method, o.c.url(o.Path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	o.c.signRequest(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != expectCode {
		return nil, newS3Error(resp)
	}
	return resp, nil
}
