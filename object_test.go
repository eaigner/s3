package s3

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var s3 = &S3{
	Bucket:    os.Getenv("S3_BUCKET"),
	AccessKey: os.Getenv("S3_KEY"),
	Secret:    os.Getenv("S3_SECRET"),
	Path:      `test`,
}

func TestDelete(t *testing.T) {
	obj := s3.Object("doesnotexist")
	// delete should always return 204 and no error
	if err := obj.Delete(); err != nil {
		t.Fatal(err)
	}
}

func TestS3(t *testing.T) {
	key := fmt.Sprintf("%d/ü n i c ö d e.txt", time.Now().UnixNano())
	o := s3.Object(key)
	o.Delete()

	// Write
	s := "hello!"
	w := o.Writer()
	_, err := io.Copy(w, bytes.NewBufferString(s))
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

	// Test access with pre-signed url
	u, err := o.ExpiringURL(60 * time.Second)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(u.String())

	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatal(resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if x := string(body); x != s {
		t.Fatal(x)
	}

	// Unauthorized access
	u.RawQuery = ""
	resp, err = http.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 403 {
		t.Fatal(resp.StatusCode)
	}

	// Head
	h, err := s3.Object(key).Head()
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
	fileName := "ü n i c ö d e.txt"
	content := "form"
	o := s3.Object(fmt.Sprintf("/%d/%s", time.Now().UnixNano(), fileName))

	p := make(Policy)
	p.SetExpiration(3600)
	p.Conditions().Bucket(s3.Bucket)
	p.Conditions().ACL(PublicRead)
	p.Conditions().Equals("$key", o.Key())

	u, err := o.FormURL(PublicRead, p)
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
			if v[0] != o.Key() {
				t.Log(o.Key())
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

	// Upload
	t.Log(u.String())

	// Assemble multipart body
	var buf bytes.Buffer
	var w = multipart.NewWriter(&buf)

	q := u.Query()
	for k, _ := range q {
		w.WriteField(k, q.Get(k))
	}
	fw, err := w.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(fw, strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	w.Close()

	// Create request
	req, err := http.NewRequest("POST", u.Scheme+`://`+u.Host, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Send
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if x := res.StatusCode; x != 204 {
		b, _ := ioutil.ReadAll(res.Body)
		t.Log(string(b))
		t.Fatal(x)
	}

	// Exists?
	exists, err := o.Exists()
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("file not found")
	}
}
