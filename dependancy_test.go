package lensvm

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleResolveModule(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	// ctx := context.Background()
	// context.Context.
	// ctx2 := vm.ResolveContext
	// cancel, ctx := vm.NewResolverContext()
	mod, err := vm.ResolveModule("file://testdata/simple/module.json")
	assert.NoError(t, err)

	assert.Equal(t, "rename", mod.Name)
	assert.Equal(t, "file://testdata/simple/main.wasm", mod.PackagePath)

	buf, err := ioutil.ReadFile("testdata/simple/main.wasm")
	assert.NoError(t, err)
	assert.Equal(t, buf, mod.PackageBytes)

}

func TestMultiResolveModule(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	mod, err := vm.ResolveModule("file://testdata/multi/module.json")
	assert.NoError(t, err)

	assert.Empty(t, mod.Name)
	assert.Len(t, mod.Modules, 2)

	buf, err := ioutil.ReadFile("testdata/multi/main1.wasm")
	if err != nil {
		panic(err)
	}

	assert.Equal(t, buf, mod.Modules[0].PackageBytes)
}

func TestImportResolveModule(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	mod, err := vm.ResolveModule("file://testdata/importsimple/module.json")
	assert.NoError(t, err)

	assert.Equal(t, "extract", mod.Name)
	assert.Len(t, mod.Import, 1)

	renameMod, ok := mod.Import["rename"]
	assert.True(t, ok)
	assert.Equal(t, "rename", renameMod.Module.Name)
}

func TestImportDeepResolveModule(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	mod, err := vm.ResolveModule("file://testdata/importdeep/module.json")
	assert.NoError(t, err)

	assert.Equal(t, "copy", mod.Name)
	assert.Len(t, mod.Import, 2)

	assert.Equal(t, "rename", mod.Import["rename"].Module.Name)
	assert.Equal(t, "file://testdata/simple/module.json", mod.Import["extract"].Module.Import["rename"].Path)
}

func TestSimpleAddModule(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	err := vm.ImportModuleFunction("rename", "file://testdata/simple/module.json")
	assert.NoError(t, err)

	assert.Len(t, vm.moduleImports, 1)
}

/*
vm := lensvm.NewVM(nil)
vm.LoadLens("...")
vm.Init()
out, err := vm.Exec
vm.ExecFunc(input, args, "rename", "file://testdata/simple/module.json")

ResolveModule("npm://lensm-vm-rename@1.4.1")
ResolveModule("wapm://lens-vm-rename@1.3.1")
ResolveModule("ipfs://Qm12384917487y139489f")

*/
