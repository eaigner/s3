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
	o.Delete()

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

	// Exists?
	exists, err := o.Exists()
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal(exists)
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

	// Head
	h, err := o.Head()
	if err != nil {
		t.Fatal(err)
	}
	if x, _ := h.ContentLength(); x != 6 {
		t.Fatal(x)
	}
	if x := h.ContentType(); x != "text/plain" {
		t.Fatal(x)
	}
	if x := h.ETag(); x == "" {
		t.Fatal(x)
	}

	// Delete
	err = o.Delete()
	if err != nil {
		t.Fatal(err)
	}

	// Exists?
	exists, err = o.Exists()
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal(exists)
	}
}

func TestFormURL(t *testing.T) {
	o := s3.Object("form.txt")

	p := make(Policy)
	p.SetExpiration(3600)
	p.Conditions().AddBucket(s3.Bucket)
	p.Conditions().AddACL(PublicRead)
	p.Conditions().MatchStartsWith("$key", "form.txt")

	u, err := o.FormUploadURL(PublicRead, p)
	if err != nil {
		t.Fatal(err)
	}

	// TODO(erik): this test isn't good but sufficient for now, enhance it
	for k, v := range u.Query() {
		switch k {
		case "AWSAccessKeyId":
			if len(v[0]) == 0 {
				t.Fatal("access key missing")
			}
		// AKIAIPXCDLRVA67BZQWA
		case "acl":
			if v[0] != "public-read" {
				t.Fatal(v)
			}
		case "key":
			if v[0] != "form.txt" {
				t.Fatal(v)
			}
		case "policy":
			if v[0] == "" {
				t.Fatal("policy missing")
			}
		case "signature":
			if len(v[0]) == 0 {
				t.Fatal("signature missing")
			}
		default:
			t.Fatal("unexpected key")
		}
	}
}
