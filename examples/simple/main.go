package main

import (
	"fmt"

	lensvm "github.com/lens-vm/lens-vm-go-host"
)

type ModuleFileDefinition struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Url         string            `json:"url,omitempty"`
	Arguments   GenericDefinition `json:"arguments"`
	Import      ImportDefinition  `json"import"`
	Runtime     string            `json:"runtime"`
	Language    string            `json:"language"`
	Module      string            `json:"module"`

	// Modules contains a list of ModuleFileDefinitions
	// if this ModuleFile defines a collection of
	// lens modules, and not just one.
	// This is only used if the rest of the parameters are
	// empty to ensure that this definition file
	// is EITHER one module or a list of modules.
	Modules []ModuleFileDefinition `json:"modules"`
}

type LensFileDefinition struct {
	Import ImportDefinition    `json:"import"`
	Lenses []GenericDefinition `json:"lenses"`
}

type ImportDefinition map[string]string
type GenericDefinition map[string]interface{}

func main() {
	vm, err := lensvm.NewHostVM(lesnvm.HostOptions{})
	if err != nil {
		panic(err)
	}

	// mod := h.NewModule([]byte("wasmBytes..."))
	mod, err := vm.LoadModule(lensvm.NewFileLoader(""))

	lens := lensvm.LensFileLoader(h, "lensfile.json")
	lens := lensvm.LensBytesLoader(h, []byte("lensfile data"))
	lens := vm.LoadLens(lensvm.LensLoader("https://"))

	out, err := vm.Exec("input document")
	if err != nil {
		panic(err)
	}
	fmt.Println("Transformed document:", out)
}
