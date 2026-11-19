package scripting

import (
	"testing"

	"github.com/dop251/goja"
)

func TestApigentAliasesGurlScriptObject(t *testing.T) {
	vm := goja.New()
	e := &Engine{vm: vm}
	RegisterGlobals(vm, e)
	v, err := vm.RunString(`apigent === gurl && typeof apigent.setVar === "function"`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.ToBoolean() {
		t.Fatal("apigent should be the same object as gurl in scripts")
	}
}
