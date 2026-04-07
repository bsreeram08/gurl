package client

import (
	"net/http"
	"time"
)

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
	Size       int64
}
