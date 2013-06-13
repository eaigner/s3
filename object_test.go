package s3

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

var s3 = &S3{
	Bucket: os.Getenv("S3_BUCKET"),
	Key:    os.Getenv("S3_KEY"),
	Secret: os.Getenv("S3_SECRET"),
}

func TestS3(t *testing.T) {
	o := s3.Object("test.txt")

	// Write
	w, err := o.Writer()
	if err != nil {
		t.Fatal(err)
	}

	s := "hello!"
	_, err = io.Copy(w, bytes.NewBufferString(s))
	if err != nil {
		t.Fatal(err)
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Read
	r, _, err := o.Reader()
	if err != nil {
		t.Fatal(err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if x := string(b); x != s {
		t.Fatal(x)
	}

	// Delete
	err = o.Delete()
	if err != nil {
		t.Fatal(err)
	}
}
