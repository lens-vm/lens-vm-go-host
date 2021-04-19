package lensvm

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleResolveModule(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	ctx := context.Background()
	c
	mod, err := vm.ResolveModule(ctx, "wapm://testdata/simple/module.json")
	assert.NoError(t, err)

	assert.Equal(t, "rename", mod.Name)
	assert.Equal(t, "file://testdata/simple/main.wasm", mod.PackagePath)

	buf, err := ioutil.ReadFile("testdata/simple/main.wasm")
	assert.NoError(t, err)
	assert.Equal(t, buf, mod.PackageBytes)
}

func TestSimpleAddModule(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	err := vm.ImportModuleFunction("rename", "file://testdata/simple/module.json")
	assert.NoError(t, err)

	assert.Len(t, vm.moduleImports, 1)
	assert.Len(t, vm.lensImports, 1)
}
