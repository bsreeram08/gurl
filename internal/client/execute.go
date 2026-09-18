package client

import (
	"context"
	"net/http"
)

func Execute(req Request) (Response, error) {
	return defaultClient.Execute(req)
}

func ExecuteWithContext(ctx context.Context, req Request) (Response, error) {
	return defaultClient.ExecuteWithContext(ctx, req)
}

var defaultClient = NewClient()

func SetDefaultCookieJar(jar http.CookieJar) {
	defaultClient.Jar = jar
}
