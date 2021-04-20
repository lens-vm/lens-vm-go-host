package lensvm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/wasmerio/wasmer-go/wasmer"

	"github.com/lens-vm/lens-vm-go-host/resolvers"
	"github.com/lens-vm/lens-vm-go-host/resolvers/file"
	"github.com/lens-vm/lens-vm-go-host/types"
)

var (
	DefaultOptions = &Options{
		Resolvers: []resolvers.Resolver{
			file.FileResolver{},
		},
	}
)

type Module struct {
	id         string
	definition types.ResolvedModule

	wmod  *wasmer.Module
	winst *wasmer.Instance
}

type Options struct {
	Resolvers     []resolvers.Resolver
	ContextValues ContextValueOptions
}

// ContextValueOptions is an option struct
// to use when creating a VM instance.
// It sets the default values to the respective
// internal contexts of the VM.
type ContextValueOptions struct {
	Resolver  map[interface{}]interface{}
	Execution map[interface{}]interface{}
}

// VM is the runtime virtual machine for
// executing Lenses
type VM struct {
	lensFile types.LensFile

	// moduleImports is a map of ID -> Module
	// where the ID is the unique identifier
	// for that module
	moduleImports map[string]*Module

	// lensImports is a map of lensName -> Module
	// where lensName is the name of a imported
	// lens function from a module
	lensImports map[string]*Module

	wengine *wasmer.Engine
	wstore  *wasmer.Store

	resolvers map[string]resolvers.Resolver

	resolverCtx context.Context
	execCtx     context.Context

	dgraph dependancyGraph

	initialized bool
}

func NewVM(opt *Options) *VM {
	if opt == nil {
		opt = DefaultOptions
	}
	wengine := wasmer.NewEngine()
	wstore := wasmer.NewStore(wengine)
	vm := &VM{
		wengine:       wengine,
		wstore:        wstore,
		moduleImports: make(map[string]*Module),
		lensImports:   make(map[string]*Module),
		resolvers:     make(map[string]resolvers.Resolver),
	}

	vm.initResolvers(opt.Resolvers)
	return vm
}

// initResolvers sets up all the given resolvers into the
// internal resolver map on the VM instance
func (vm *VM) initResolvers(res []resolvers.Resolver) {
	for _, r := range res {
		scheme := r.Scheme()
		if scheme == "" {
			panic("Resolver is missing a scheme, cannot be empty")
		}

		if _, exists := vm.resolvers[scheme]; exists {
			panic("Duplicate resolver scheme")
		}
		vm.resolvers[scheme] = r
	}
}

func (vm *VM) LoadLens(l LensLoader) error {
	ctx := context.TODO()
	lens, err := l.Load(ctx)
	if err != nil {
		return err
	}
	vm.lensFile = lens
	return vm.resolveLens(ctx, vm.lensFile)
}

// Init initializes the virtual machine, assuming it as a loaded
// lens file object. It creates the underlying WASM module
// instances, and dynamically links all dependancies
func (vm *VM) Init() error {
	return nil
}

// Exec does the actual lens execution and transformation
// of the input, producing some output. It will execute all
// the lenses in the LensFile, incrementally merging the
// individual outputs, until it completes all lenses.
func (vm *VM) Exec(input []byte) (out []byte, err error) {
	return
}

// func (vm *VM) ResolverContext()

func (vm *VM) resolveLens(ctx context.Context, lens types.LensFile) error {
	foundModules := make(map[string]bool)
	for name, modPath := range lens.Import {
		resolvedMod, err, ok := vm.resolveModule(ctx, foundModules, modPath)
		if err != nil {
			return err
		}

		if !ok {
			continue
		}

		if err := vm.addModuleImportReference(name, resolvedMod); err != nil {
			return err
		}
	}
	return nil
}

/*

vm := host.NewVM(...)
vm.AddModule(lensvm.ModuleIPFSLoader("ipfs://"))

*/

// ImportModule will add the module file found
// when resolving the given path. It then adds all the
// defined lens modules in the module file.
func (vm *VM) ImportModule(path string) error {
	return nil
}

// ImportModuleFunction will resolve the module from the
// given path, and will import only the named module.
// If the Module file contains an array of defined modules
// it will scan all of them until it finds a match. If
// the module only defines a single module definition, it
// will import it if theres a name match.
func (vm *VM) ImportModuleFunction(name, path string) error {
	rmod, err := vm.ResolveModule(path)
	if err != nil {
		return err
	}

	return vm.addModuleImportReference(name, rmod)
}

// func (vm) ResolveContext()

func (vm *VM) addModuleImportReference(name string, rmod types.ResolvedModule) error {
	if len(name) == 0 {
		return errors.New("Missing name of import function")
	}
	if len(rmod.ID) == 0 {
		return errors.New("Invalud resolved module object. Missing ID")
	}
	if mod, exists := vm.moduleImports[rmod.ID]; exists {
		if moduleHasLensFunc(rmod, name) {
			vm.lensImports[name] = mod
			return nil
		}
		return fmt.Errorf("Lens function '%s' is missing from module", name)
	}

	mod, err := vm.newModule(rmod)
	if err != nil {
		return err
	}

	vm.moduleImports[mod.id] = mod
	vm.lensImports[name] = mod
	return nil
}

// func (vm *VM) HasImport

// func (vm *VM) AddResolvedModule(rmod types.ResolvedModule) error {

// }

func (vm *VM) newModule(rmod types.ResolvedModule) (*Module, error) {
	// check ID and PackageBytes
	if rmod.ID == "" || len(rmod.PackageBytes) == 0 {
		return nil, errors.New("Invalid resolved module object. Missing ID or PackageBytes")
	}
	mod := &Module{
		id:         rmod.ID,
		definition: rmod,
	}

	wmod, err := wasmer.NewModule(vm.wstore, rmod.PackageBytes)
	if err != nil {
		return nil, err
	}
	mod.wmod = wmod
	return mod, nil
}

func (vm *VM) ResolveModule(path string) (types.ResolvedModule, error) {
	foundModules := make(map[string]bool)
	// preload the found map with our current imports
	for k, _ := range vm.moduleImports {
		foundModules[k] = true
	}
	ctx := context.TODO()
	mod, err, _ := vm.resolveModule(ctx, foundModules, path)
	return mod, err
}

// resolveModule takes a path and map of previous resolved modules
// and returns the module along with its imports resolved.
// If it imports an already resolved module, that import will
// contain an empty ResolvedModule type.
func (vm *VM) resolveModule(ctx context.Context, foundModules map[string]bool, path string) (types.ResolvedModule, error, bool) {
	if foundModules == nil {
		foundModules = make(map[string]bool)
	}
	if _, ok := foundModules[path]; ok {
		return types.ResolvedModule{}, nil, false
	}
	foundModules[path] = true

	buf, err := vm.resolve(ctx, path)
	if err != nil {
		return types.ResolvedModule{}, err, false
	}

	var modFile types.ModuleFile
	err = json.Unmarshal(buf, &modFile)
	if err != nil {
		return types.ResolvedModule{}, err, false
	}

	//validate ModuleFile
	if modFile.Name != "" && len(modFile.Modules) > 0 {
		return types.ResolvedModule{}, fmt.Errorf("Resolved module at path %s connot have a single module and a 'modules' array", path), false
	}

	var rMods []types.ResolvedModule
	// flatten the original modFile into a array.
	// It either contains the original modFile.Modules array
	// or an array of length 1, which is the root module in the
	// modFile
	if len(modFile.Modules) == 0 {
		modFile.Modules = append(modFile.Modules, modFile)
	}
	for _, f := range modFile.Modules {
		m := types.ModuleToResolvedModule(f)
		for n, p := range f.Import {
			// impModule := types.ImportedModule{Path: p}
			mod, err, _ := vm.resolveModule(ctx, foundModules, p)
			if err != nil {
				return types.ResolvedModule{}, err, false
			}
			m.Import[n] = types.ImportedModule{
				Path:   p,
				Module: mod,
			}
		}

		wasmBytes, err := vm.resolve(ctx, f.Package)
		if err != nil {
			return types.ResolvedModule{}, err, false
		}
		m.PackageBytes = wasmBytes
		m.ID = path

		rMods = append(rMods, m)
	}

	// return either a root resolved module
	// or a resolved module with an array
	// of sub resolved modules
	if len(rMods) == 1 {
		return rMods[0], nil, true
	}
	// @todo: Should we add the extra resolver meta-data here?
	return types.ResolvedModule{
		Modules: rMods,
	}, nil, true
}

func (vm *VM) resolve(ctx context.Context, path string) ([]byte, error) {
	if !strings.Contains(path, "://") {
		return nil, errors.New("resolve path is missing a URI scheme")
	}
	parts := strings.Split(path, "://")
	if len(parts) != 2 {
		return nil, errors.New("Malformed resolve path, must be of the form '<scheme>://<path>'")
	}

	resolver, ok := vm.resolvers[parts[0]]
	if !ok {
		return nil, fmt.Errorf("No resolver for given scheme %s", parts[0])
	}

	return resolver.Resolve(ctx, parts[1])
}

func hash256(buf []byte) string {
	h := sha256.New()
	h.Write(buf)
	return string(h.Sum(nil))
}

func moduleHasLensFunc(rmod types.ResolvedModule, name string) bool {
	if len(name) == 0 {
		return false
	}

	if name == rmod.Name {
		return true
	}

	for _, mod := range rmod.Modules {
		if name == mod.Name {
			return true
		}
	}

	return false
}

func isEmptyResolvedModule(mod types.ResolvedModule) bool {
	return (mod.Name == "" && len(mod.Modules) == 0)
}
