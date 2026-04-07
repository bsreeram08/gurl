package client

func Execute(req Request) (Response, error) {
	return defaultClient.Execute(req)
}

var defaultClient = NewClient()
