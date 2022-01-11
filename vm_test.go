package lensvm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVMInitSimpleLens(t *testing.T) {
	vm := NewVM(nil)
	assert.NotNil(t, vm)

	err := vm.LoadLens(LensFileLoader("file://testdata/lens/simple/lens.json"))
	assert.NoError(t, err)

	err = vm.Init()
	assert.NoError(t, err)

	for k, v := range vm.lensImports {
		assert.NotNil(t, v.wmod, k)
		assert.NotNil(t, v.winst, k)
	}

	for k, v := range vm.moduleImports {
		assert.NotNil(t, v.wmod, k)
		assert.NotNil(t, v.winst, k)
	}
}
