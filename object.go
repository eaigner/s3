package s3

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

type WriteAbortCloser interface {
	io.WriteCloser
	Abort() error
}

type Object struct {
	c   *S3
	Key string
}

type ACL string

const (
	Private           = ACL("private")
	PublicRead        = ACL("public-read")
	PublicReadWrite   = ACL("public-read-write")
	AuthenticatedRead = ACL("authenticated-read")
	BucketOwnerRead   = ACL("bucket-owner-read")
	BucketOwnerFull   = ACL("bucket-owner-full-control")
)

// FormUpload returns a new signed form upload object
func (o *Object) FormUploadURL(acl ACL, policy Policy, customParams ...url.Values) (*url.URL, error) {
	b, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	policy64 := base64.StdEncoding.EncodeToString(b)
	mac := hmac.New(sha1.New, []byte(o.c.Secret))
	mac.Write([]byte(policy64))

	u := o.c.url("")
	val := make(url.Values)
	val.Set("AWSAccessKeyId", o.c.Key)
	val.Set("acl", string(acl))
	val.Set("key", o.Key)
	val.Set("signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	val.Set("policy", policy64)
	for _, p := range customParams {
		for k, v := range p {
			for _, v2 := range v {
				val.Add(k, v2)
			}
		}
	}

	u.RawQuery = val.Encode()

	return u, nil
}

// Delete deletes the S3 object.
func (o *Object) Delete() error {
	_, err := o.request("DELETE", 204)
	return err
}

// Exists tests if an object already exists.
func (o *Object) Exists() (bool, error) {
	resp, err := o.request("HEAD", 0)
	if err != nil {
		return false, err
	}
	return (resp.StatusCode == 200), nil
}

// Writer returns a new WriteAbortCloser you can write to.
// The written data will be uploaded as a multipart request.
func (o *Object) Writer() (WriteAbortCloser, error) {
	return newUploader(o.c, o.Key)
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
	req, err := http.NewRequest(method, o.c.url(o.Key).String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	o.c.signRequest(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if expectCode != 0 && resp.StatusCode != expectCode {
		return nil, newS3Error(resp)
	}
	return resp, nil
}

type Policy map[string]interface{}

func (p Policy) SetExpiration(seconds uint) {
	p["expiration"] = time.Now().UTC().Add(time.Second * time.Duration(seconds)).Format("2006-01-02T15:04:05Z")
}

func (p Policy) Conditions() *PolicyConditions {
	key := "conditions"
	if _, ok := p[key]; !ok {
		pol := make(PolicyConditions, 0, 5)
		p[key] = &pol
	}
	if t, ok := p[key].(*PolicyConditions); ok {
		return t
	}
	panic("unreachable")
}

type PolicyConditions []interface{}

func (c *PolicyConditions) Add(key, value string) {
	*c = append(*c, map[string]string{key: value})
}

func (c *PolicyConditions) AddBucket(bucket string) {
	c.Add("bucket", bucket)
}

func (c *PolicyConditions) AddACL(acl ACL) {
	c.Add("acl", string(acl))
}

func (c *PolicyConditions) AddRedirect(url string) {
	c.Add("redirect", url)
}

func (c *PolicyConditions) AddSuccessActionRedirect(url string) {
	c.Add("success_action_redirect", url)
}

func (c *PolicyConditions) Match(mtype, cond, match string) {
	*c = append(*c, []string{mtype, cond, match})
}

func (c *PolicyConditions) MatchEquals(cond, match string) {
	c.Match("eq", cond, match)
}

func (c *PolicyConditions) MatchStartsWith(cond, match string) {
	c.Match("starts-with", cond, match)
}
