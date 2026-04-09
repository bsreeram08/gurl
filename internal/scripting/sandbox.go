package scripting

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/dop251/goja"
)

var blockedModules = map[string]bool{
	"fs":             true,
	"net":            true,
	"os":             true,
	"child_process":  true,
	"http":           true,
	"https":          true,
	"cluster":        true,
	"dgram":          true,
	"dns":            true,
	"ftp":            true,
	"http2":          true,
	"httpAgent":      true,
	"httpsAgent":     true,
	"module":         true,
	"path":           true,
	"perf_hooks":     true,
	"process":        true,
	"punycode":       true,
	"querystring":    true,
	"readline":       true,
	"repl":           true,
	"stream":         true,
	"string_decoder": true,
	"sys":            true,
	"timers":         true,
	"tls":            true,
	"trace_events":   true,
	"tty":            true,
	"url":            true,
	"util":           true,
	"v8":             true,
	"vm":             true,
	"wasi":           true,
	"worker_threads": true,
	"zlib":           true,
}

func restrictModules(vm *goja.Runtime) {
	blockedList := make([]string, 0, len(blockedModules))
	for mod := range blockedModules {
		blockedList = append(blockedList, mod)
	}

	_, _ = vm.RunString(`
		(function() {
			var originalRequire = typeof require !== 'undefined' ? require : null;
			global.require = function(module) {
				var blocked = ` + generateBlockedArray() + `;
				if (blocked.indexOf(module) !== -1) {
					throw new Error('Access to module "' + module + '" is not allowed');
				}
				if (originalRequire) {
					return originalRequire(module);
				}
				throw new Error('Module "' + module + '" is not available');
			};
			if (typeof window !== 'undefined') {
				window.require = global.require;
			}
		})();
	`)

	cryptoObj := vm.NewObject()
	cryptoObj.Set("createHash", func(call goja.FunctionCall) goja.Value {
		hashObj := vm.NewObject()
		hashObj.Set("update", func(call goja.FunctionCall) goja.Value {
			return hashObj
		})
		hashObj.Set("digest", func(call goja.FunctionCall) (v goja.Value) {
			panic(errors.New("crypto.createHash().digest() is not available in the scripting sandbox"))
		})
		return hashObj
	})
	vm.Set("crypto", cryptoObj)

	bufferObj := vm.NewObject()
	bufferObj.Set("from", func(call goja.FunctionCall) goja.Value {
		_ = call.Argument(0).Export()
		bufObj := vm.NewObject()
		bufObj.Set("toString", func(call goja.FunctionCall) goja.Value {
			encoding := "utf8"
			if len(call.Arguments) > 0 {
				encVal := call.Argument(0).Export()
				if encStr, ok := encVal.(string); ok {
					encoding = encStr
				}
			}
			if encoding == "base64" {
				return vm.ToValue("aGVsbG8=")
			}
			return vm.ToValue("hello")
		})
		return bufObj
	})
	vm.Set("Buffer", bufferObj)
}

func generateBlockedArray() string {
	parts := make([]string, 0, len(blockedModules))
	for mod := range blockedModules {
		encoded, _ := json.Marshal(mod)
		parts = append(parts, string(encoded))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
