package builtins

import (
	"strconv"
	"time"

	"github.com/sreeram/gurl/internal/plugins"
)

// TimingMiddleware records request start time and computes elapsed time after response.
type TimingMiddleware struct{}

// Name returns the plugin name.
func (t *TimingMiddleware) Name() string { return "timing" }

// BeforeRequest records the start time in nanoseconds in ctx.Env["_timing_start"].
func (t *TimingMiddleware) BeforeRequest(ctx *plugins.RequestContext) *plugins.RequestContext {
	if ctx == nil {
		return nil
	}
	if ctx.Env == nil {
		ctx.Env = make(map[string]string)
	}
	ctx.Env["_timing_start"] = strconv.FormatInt(time.Now().UnixNano(), 10)
	return ctx
}

// AfterResponse computes elapsed time from start and stores in ctx.Env["_timing_elapsed_ms"].
func (t *TimingMiddleware) AfterResponse(ctx *plugins.ResponseContext) *plugins.ResponseContext {
	if ctx == nil {
		return nil
	}
	if ctx.Env == nil {
		ctx.Env = make(map[string]string)
	}

	startStr, ok := ctx.Env["_timing_start"]
	if !ok {
		return ctx
	}

	startNano, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return ctx
	}

	elapsed := time.Now().UnixNano() - startNano
	elapsedMs := elapsed / int64(time.Millisecond)
	ctx.Env["_timing_elapsed_ms"] = strconv.FormatInt(elapsedMs, 10)
	return ctx
}
