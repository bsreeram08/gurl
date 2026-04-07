package client

import "net/http"

func Execute(req Request) (Response, error) {
	return defaultClient.Execute(req)
}

var defaultClient = NewClient()

func SetDefaultCookieJar(jar http.CookieJar) {
	defaultClient.Jar = jar
}
