package client

import (
	"net/http"
	"time"
)

type RedirectHop struct {
	URL        string
	StatusCode int
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
	Size       int64
	Redirects  []RedirectHop
}
