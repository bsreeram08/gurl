package scripting

import (
	"encoding/json"
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

	_, _ = vm.RunString(`
		(function() {
			var originalEval = eval;
			eval = function() {
				throw new Error("eval is not allowed in sandbox");
			};
			if (typeof window !== 'undefined') {
				window.eval = eval;
			}
		})();
	`)

	_, _ = vm.RunString(`
		(function() {
			var originalFunction = Function;
			Function = function() {
				throw new Error("Function is not allowed in sandbox");
			};
			if (typeof window !== 'undefined') {
				window.Function = Function;
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
			panic(vm.NewTypeError("crypto.createHash().digest() is not available in the scripting sandbox"))
		})
		return hashObj
	})
	vm.Set("crypto", cryptoObj)

	bufferObj := vm.NewObject()
	bufferObj.Set("from", func(call goja.FunctionCall) goja.Value {
		panic(vm.NewTypeError("Buffer.from is not available in the scripting sandbox"))
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
