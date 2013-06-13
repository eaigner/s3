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

type S3 struct {
	Bucket string
	Key    string
	Secret string
}

// Object returns a new S3 object handle at path.
func (c *S3) Object(path string) *Object {
	return &Object{
		c:    c,
		Path: path,
	}
}

func (c *S3) url(path string) string {
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", c.Bucket, path)
}

func (c *S3) signRequest(req *http.Request) {
	amzHeaders := ""
	resource := "/" + c.Bucket + req.URL.Path

	// Parameters require specific ordering
	query := req.URL.Query()
	if len(query) > 0 {
		keys := []string{}
		for k := range query {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		parts := []string{}
		for _, key := range keys {
			vals := query[key]
			for _, val := range vals {
				if val == "" {
					parts = append(parts, url.QueryEscape(key))
				} else {
					part := fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(val))
					parts = append(parts, part)
				}
			}
		}

		req.URL.RawQuery = strings.Join(parts, "&")
	}

	if req.URL.RawQuery != "" {
		resource += "?" + req.URL.RawQuery
	}

	if req.Header.Get("Date") == "" {
		req.Header.Set("Date", time.Now().Format(time.RFC1123))
	}

	authStr := strings.Join([]string{
		strings.TrimSpace(req.Method),
		req.Header.Get("Content-MD5"),
		req.Header.Get("Content-Type"),
		req.Header.Get("Date"),
		amzHeaders + resource,
	}, "\n")

	h := hmac.New(sha1.New, []byte(c.Secret))
	h.Write([]byte(authStr))

	h64 := base64.StdEncoding.EncodeToString(h.Sum(nil))
	auth := "AWS" + " " + c.Key + ":" + h64
	req.Header.Set("Authorization", auth)
}
