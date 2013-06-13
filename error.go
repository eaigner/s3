package s3

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type s3error struct {
	resp *http.Response
	body bytes.Buffer
}

func newS3Error(r *http.Response) *s3error {
	err := &s3error{resp: r}
	io.Copy(&err.body, r.Body)
	r.Body.Close()
	return err
}

func (e *s3error) Error() string {
	return fmt.Sprintf("s3 returned status %d (%s)", e.resp.StatusCode, e.body.String())
}
