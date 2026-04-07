package client

import (
	"time"
)

type Header struct {
	Key   string
	Value string
}

type Request struct {
	Method  string
	URL     string
	Headers []Header
	Body    string
	Timeout time.Duration
}

var defaultTimeout = 30 * time.Second
