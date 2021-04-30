package lensvm

import (
	"fmt"
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
	assert.Len(t, mod.Exports, 2)

	buf, err := ioutil.ReadFile("testdata/multi/main1.wasm")
	if err != nil {
		panic(err)
	}

	assert.Equal(t, buf, mod.PackageBytes)
}

func TestImportSimpleResolveModule(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	mod, err := vm.ResolveModule("file://testdata/importsimple/module.json")
	assert.NoError(t, err)

	assert.Equal(t, "extract", mod.Name)
	assert.Len(t, mod.Imports, 1)

	renameMod, ok := mod.Imports["rename"]
	assert.True(t, ok)
	assert.Equal(t, "rename", renameMod.Module.Name)
}

func TestImportDeepResolveModule(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	mod, err := vm.ResolveModule("file://testdata/importdeep/module.json")
	assert.NoError(t, err)

	assert.Equal(t, "copy", mod.Name)
	assert.Len(t, mod.Imports, 2)

	assert.Equal(t, "rename", mod.Imports["rename"].Module.Name)
	assert.Equal(t, "file://testdata/simple/module.json", mod.Imports["extract"].Module.Imports["rename"].Path)
}

func TestSimpleImportFunction(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	_, err := vm.ImportModuleFunction("rename", "file://testdata/simple/module.json")
	assert.NoError(t, err)

	assert.Len(t, vm.moduleImports, 1)
}

func TestMultiImportFunction(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	_, err := vm.ImportModuleFunction("rename", "file://testdata/multi/module.json")
	assert.NoError(t, err)

	assert.Len(t, vm.moduleImports, 1)
}

func TestDeepImportFunction(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	_, err := vm.ImportModuleFunction("rename", "file://testdata/importdeep/module.json")
	assert.NoError(t, err)

	assert.Len(t, vm.moduleImports, 3, "stuff")
}

func TestSimpleLens(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	err := vm.LoadLens(LensFileLoader("file://testdata/lens/simple/lens.json"))
	assert.NoError(t, err)

	assert.Len(t, vm.moduleImports, 1)
	assert.Len(t, vm.lensImports, 1)
}

func TestDeepLens(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	err := vm.LoadLens(LensFileLoader("file://testdata/lens/importdeep/lens.json"))
	assert.NoError(t, err)

	assert.Len(t, vm.moduleImports, 3)
	fmt.Println(vm.lensImports)
	assert.Len(t, vm.lensImports, 1)
}

func TestGraphSortSimple(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	err := vm.LoadLens(LensFileLoader("file://testdata/lens/simple/lens.json"))
	assert.NoError(t, err)

	err = vm.makeDependancyGraph()
	assert.NoError(t, err)

	var i string
	// grab the first import path
	for _, v := range vm.lensFile.Import {
		i = v
		break
	}
	deps, err := vm.dgraph.SortOrder(i)
	assert.NoError(t, err)

	assert.Len(t, deps, 1)
	assert.Equal(t, "file://testdata/simple/module.json", deps[0])
}

func TestGraphSortDeep(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	err := vm.LoadLens(LensFileLoader("file://testdata/lens/importdeep/lens.json"))
	assert.NoError(t, err)

	err = vm.makeDependancyGraph()
	assert.NoError(t, err)

	var i string
	// grab the first import path
	for _, v := range vm.lensFile.Import {
		i = v
		break
	}
	deps, err := vm.dgraph.SortOrder(i)
	assert.NoError(t, err)

	assert.Len(t, deps, 3)
	assert.Equal(t, []string{
		"file://testdata/simple/module.json",
		"file://testdata/importsimple/module.json",
		"file://testdata/importdeep/module.json"},
		deps)
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
