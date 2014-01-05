package s3

import (
	"net/http"
	"strings"
	"testing"
)

func TestSignRequest(t *testing.T) {
	// use unicode values in url
	req, err := http.NewRequest("GET", "https://bücket/päth/këy?a&c=y&b=ö", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add(`Content-MD5`, `md5`)
	req.Header.Add(`Content-Type`, `ctype`)
	req.Header.Add(`Date`, `date`)
	req.Header.Add(`x-amz-a`, `x`)
	req.Header.Add(`x-amz-a`, `y`)
	req.Header.Add(`x-amz-b`, `z`)

	s3 := &S3{
		AccessKey: "s3key",
		Secret:    "s3secret",
	}

	// check auth string
	s := s3.authString(req)
	c := strings.Split(s, "\n")

	if x := len(c); x != 7 {
		t.Fatal(x)
	}
	if x := c[0]; x != "GET" {
		t.Fatal(x)
	}
	if x := c[1]; x != "md5" {
		t.Fatal(x)
	}
	if x := c[2]; x != "ctype" {
		t.Fatal(x)
	}
	if x := c[3]; x != "date" {
		t.Fatal(x)
	}
	if x := c[4]; x != "x-amz-a:x,y" {
		t.Fatal(x)
	}
	if x := c[5]; x != "x-amz-b:z" {
		t.Fatal(x)
	}
	if x := c[6]; x != `/p%C3%A4th/k%C3%ABy?a&b=%C3%B6&c=y` {
		t.Fatal(x)
	}

	// sign
	s3.signRequest(req)

	if x := req.Header.Get(`Authorization`); x != "AWS s3key:USqIVSnPtCdhZqaWq+d3cGvIrhQ=" {
		t.Fatal(x)
	}
}
