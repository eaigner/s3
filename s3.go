package s3

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// S3 holds the S3 configuration
type S3 struct {
	// Bucket is the S3 bucket to use
	Bucket string

	// AccessKey is the S3 access key
	AccessKey string

	// Secret is the S3 secret
	Secret string

	// Path is the path to prepend to all keys
	Path string
}

func (s3 *S3) Object(key string) Object {
	return &object{key: key, s3: *s3}
}

// http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html
func (s3 *S3) authString(req *http.Request) string {
	if req.Header.Get("Date") == "" {
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}

	// canonicalize amz headers
	a := make([]string, 0, 1)
	for k, _ := range req.Header {
		k = strings.ToLower(k)
		if strings.HasPrefix(k, "x-amz-") {
			a = append(a, k)
		}
	}

	sort.Strings(a)

	for i, v := range a {
		k := http.CanonicalHeaderKey(v)
		vv := req.Header[k]
		a[i] = v + `:` + strings.Join(vv, `,`) + "\n"
	}

	canonicalAmzHeaders := strings.Join(a, "")

	// canonicalize resource
	cres, rawQuery := canonicalResource(req.URL.Path, req.URL.Query())
	req.URL.RawQuery = rawQuery

	return strings.Join([]string{
		strings.TrimSpace(req.Method),
		req.Header.Get("Content-MD5"),
		req.Header.Get("Content-Type"),
		req.Header.Get("Date"),
		canonicalAmzHeaders + cres,
	}, "\n")
}

func canonicalResource(path string, query url.Values) (cres, rawQuery string) {
	p := strings.Split(path, `/`)
	for i, v := range p {
		p[i] = escape(v)
	}
	cres = strings.Join(p, `/`)

	if len(query) > 0 {
		a := make([]string, 0, 1)
		for k := range query {
			a = append(a, k)
		}

		sort.Strings(a)

		parts := make([]string, 0, len(a))
		for _, k := range a {
			vv := query[k]
			for _, v := range vv {
				if v == "" {
					parts = append(parts, escape(k))
				} else {
					parts = append(parts, fmt.Sprintf("%s=%s", escape(k), escape(v)))
				}
			}
		}

		qs := strings.Join(parts, "&")

		rawQuery = qs
		cres += `?` + qs
	}

	return
}

// escape ensures everything is properly escaped and spaces use %20 instead of +
func escape(s string) string {
	return strings.Replace(url.QueryEscape(s), `+`, `%20`, -1)
}

func (s3 *S3) signRequest(req *http.Request) {
	authStr := s3.authString(req)

	h := hmac.New(sha1.New, []byte(s3.Secret))
	h.Write([]byte(authStr))

	h64 := base64.StdEncoding.EncodeToString(h.Sum(nil))
	auth := "AWS " + s3.AccessKey + ":" + h64
	req.Header.Set("Authorization", auth)
}
