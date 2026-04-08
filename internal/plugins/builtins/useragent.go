package builtins

import (
	"strings"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/plugins"
)

// UserAgentMiddleware sets a default User-Agent header if none is provided.
type UserAgentMiddleware struct {
	Version string
}

// Name returns the plugin name.
func (u *UserAgentMiddleware) Name() string { return "user-agent" }

// BeforeRequest sets the User-Agent header to "gurl/{Version}" if not already set.
func (u *UserAgentMiddleware) BeforeRequest(ctx *plugins.RequestContext) *plugins.RequestContext {
	if ctx == nil || ctx.Request == nil {
		return ctx
	}

	if !hasHeader(ctx.Request.Headers, "User-Agent") {
		userAgent := "gurl/" + u.Version
		ctx.Request.Headers = append(ctx.Request.Headers, client.Header{
			Key:   "User-Agent",
			Value: userAgent,
		})
	}

	return ctx
}

// AfterResponse is a pass-through that returns ctx unchanged.
func (u *UserAgentMiddleware) AfterResponse(ctx *plugins.ResponseContext) *plugins.ResponseContext {
	return ctx
}

// hasHeader checks if the headers slice contains a header with the given key (case-insensitive).
func hasHeader(headers []client.Header, key string) bool {
	for _, h := range headers {
		if strings.EqualFold(h.Key, key) {
			return true
		}
	}
	return false
}
