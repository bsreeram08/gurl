package scripting

import (
	"testing"

	"github.com/dop251/goja"
)

func TestValetkeyAliasesGurlScriptObject(t *testing.T) {
	vm := goja.New()
	e := &Engine{vm: vm}
	RegisterGlobals(vm, e)
	v, err := vm.RunString(`valetkey === gurl && typeof valetkey.setVar === "function"`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.ToBoolean() {
		t.Fatal("valetkey should be the same object as gurl in scripts")
	}
}
