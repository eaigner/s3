package s3

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ACL string

const (
	Private           ACL = "private"
	PublicRead        ACL = "public-read"
	PublicReadWrite   ACL = "public-read-write"
	AuthenticatedRead ACL = "authenticated-read"
	BucketOwnerRead   ACL = "bucket-owner-read"
	BucketOwnerFull   ACL = "bucket-owner-full-control"
)

const (
	s3proto = `https`
	s3host  = `s3.amazonaws.com`
)

type Object interface {
	// Key returns the object key. If a path was specified in the S3 configuration
	// it will be prepended to the key.
	Key() string

	// S3 returns the configuration this object is bound to.
	S3() S3

	// Writer returns a new upload io.Writer
	Writer() Writer

	// Reader returns a new ReadCloser to read the file
	Reader() (io.ReadCloser, http.Header, error)

	// Exists checks if an object with the specified key already exists
	Exists() (bool, error)

	// Delete deletes an object
	Delete() error

	// Head does a HEAD request and returns the header
	Head() (Header, error)

	// ExpiringURL returns a signed, expiring URL for the object
	ExpiringURL(expiresIn time.Duration) (*url.URL, error)

	// FormURL returns a signed URL for multipart form uploads
	FormURL(acl ACL, policy Policy, query ...url.Values) (*url.URL, error)
}

type object struct {
	key string
	s3  S3
}

func (o *object) Key() string {
	if p := trim(o.s3.Path); p != "" {
		return p + `/` + trim(o.key)
	}
	return trim(o.key)
}

func (o *object) S3() S3 {
	return o.s3
}

func (o *object) Writer() Writer {
	return newWriter(o)
}

func (o *object) Reader() (io.ReadCloser, http.Header, error) {
	resp, err := o.request("GET", 200, "error creating reader")
	if err != nil {
		return nil, nil, err
	}
	return resp.Body, resp.Header, nil
}

func (o *object) Exists() (bool, error) {
	resp, err := o.request("HEAD", 0, "")
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	return (resp.StatusCode == 200), err

}

func (o *object) Delete() error {
	resp, err := o.request("DELETE", 204, "error deleting object")
	if err != nil {
		return err
	}
	resp.Body.Close()
	return err
}

func (o *object) Head() (Header, error) {
	resp, err := o.request("HEAD", 200, "error getting head")
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return Header(resp.Header), nil
}

func (o *object) ExpiringURL(expiresIn time.Duration) (*url.URL, error) {
	// create signature string
	// TODO(erik): unify this with the request signing method.
	method := "GET"
	expires := strconv.FormatInt(time.Now().Add(expiresIn).Unix(), 10)
	cres, _ := canonicalResource(o.resource(""), nil)
	toSign := method + "\n\n\n" + expires + "\n" + cres

	// generate signature
	mac := hmac.New(sha1.New, []byte(o.s3.Secret))
	mac.Write([]byte(toSign))

	sig := strings.TrimSpace(base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	// assemble url
	var v = make(url.Values)
	v.Set("AWSAccessKeyId", o.s3.AccessKey)
	v.Set("Expires", expires)
	v.Set("Signature", sig)

	u, err := url.Parse(o.url(""))
	if err != nil {
		return nil, err
	}
	u.RawQuery = v.Encode()

	return u, nil
}

func (o *object) FormURL(acl ACL, policy Policy, query ...url.Values) (*url.URL, error) {
	b, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	policy64 := base64.StdEncoding.EncodeToString(b)
	mac := hmac.New(sha1.New, []byte(o.s3.Secret))
	mac.Write([]byte(policy64))

	uv := make(url.Values)
	uv.Set("AWSAccessKeyId", o.s3.AccessKey)
	uv.Set("acl", string(acl))
	uv.Set("key", o.Key())
	uv.Set("signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	uv.Set("policy", policy64)
	for _, p := range query {
		for k, v := range p {
			for _, v2 := range v {
				uv.Add(k, v2)
			}
		}
	}

	u, err := url.Parse(s3proto + `://` + o.s3.Bucket + `.` + s3host)
	if err != nil {
		return nil, err
	}
	u.RawQuery = uv.Encode()

	return u, nil
}

func (o *object) request(method string, code int, serr string) (*http.Response, error) {
	req, err := http.NewRequest(method, o.url(""), nil)
	if err != nil {
		return nil, err
	}

	o.s3.signRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if c := resp.StatusCode; code > 0 && c != code {
		return nil, fmt.Errorf("s3: %s (%s)", serr, http.StatusText(c))
	}

	return resp, nil
}

func (o *object) resource(query string) string {
	return `/` + o.s3.Bucket + `/` + o.Key() + query
}

func (o *object) url(query string) string {
	return s3proto + `://` + s3host + o.resource(query)
}

func trim(s string) string {
	return strings.Trim(s, ` /`)
}
