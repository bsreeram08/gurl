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

// blockedArrayJS is pre-computed at init for use in restrictModules
var blockedArrayJS string

func init() {
	parts := make([]string, 0, len(blockedModules))
	for mod := range blockedModules {
		encoded, _ := json.Marshal(mod)
		parts = append(parts, string(encoded))
	}
	blockedArrayJS = "[" + strings.Join(parts, ",") + "]"
}

// registerSandboxRestricted sets up sandbox restrictions on a new runtime
// Called once per runtime when created, not on each reuse
func registerSandboxRestricted(vm *goja.Runtime) {
	if err := applySandbox(vm); err != nil {
		panic("sandbox setup failed: " + err.Error())
	}
}

func applySandbox(vm *goja.Runtime) error {
	if _, err := vm.RunString(`
		(function() {
			var originalRequire = typeof require !== 'undefined' ? require : null;
			var g = (typeof globalThis !== 'undefined') ? globalThis : this;

			g.require = function(module) {
				var blocked = ` + blockedArrayJS + `;
				if (blocked.indexOf(module) !== -1) {
					throw new Error('Access to module "' + module + '" is not allowed');
				}
				if (originalRequire) {
					return originalRequire(module);
				}
				throw new Error('Module "' + module + '" is not available');
			};

			if (typeof window !== 'undefined') {
				window.require = g.require;
			}
		})();
	`); err != nil {
		return err
	}

	vm.Set("eval", func(call goja.FunctionCall) goja.Value {
		panic(vm.NewTypeError("eval is not allowed in sandbox"))
	})

	if _, err := vm.RunString(`
		(function() {
			var blocked = function() {
				throw new Error("Function is not allowed in sandbox");
			};
			Function.prototype.constructor = blocked;
			try {
				Object.getPrototypeOf(async function () {}).constructor = blocked;
			} catch (e) {}
			try {
				Object.getPrototypeOf(function* () {}).constructor = blocked;
			} catch (e) {}
			Function = blocked;
			var g = (typeof globalThis !== 'undefined') ? globalThis : this;
			g.Function = blocked;
		})();
	`); err != nil {
		return err
	}

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
	return nil
}
