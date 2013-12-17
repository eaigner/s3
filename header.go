package s3

import (
	"net/http"
	"strconv"
	"time"
)

type Header http.Header

func (h Header) Date() (time.Time, error) {
	return time.Parse(time.RFC1123, http.Header(h).Get("Date"))
}

func (h Header) LastModified() (time.Time, error) {
	return time.Parse(time.RFC1123, http.Header(h).Get("Last-Modified"))
}

func (h Header) ETag() string {
	return http.Header(h).Get("ETag")
}

func (h Header) ContentLength() (int64, error) {
	return strconv.ParseInt(http.Header(h).Get("Content-Length"), 10, 64)
}

func (h Header) ContentType() string {
	return http.Header(h).Get("Content-Type")
}
