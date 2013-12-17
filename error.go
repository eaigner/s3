package s3

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

type S3Error struct {
	statusCode int
	body       string
}

func newS3Error(r *http.Response) *S3Error {
	defer r.Body.Close()
	err := &S3Error{statusCode: r.StatusCode}
	// copy xml error description body
	b, _ := ioutil.ReadAll(r.Body)
	err.body = string(b)
	return err
}

func (e *S3Error) Error() string {
	return fmt.Sprintf("s3: %d", e.statusCode)
}

func (e *S3Error) StatusCode() int {
	return e.statusCode
}

func (e *S3Error) XMLBody() string {
	return e.body
}
