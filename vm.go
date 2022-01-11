package lensvm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/lens-vm/lens-vm-go-host/resolvers"
	"github.com/lens-vm/lens-vm-go-host/resolvers/file"
	"github.com/lens-vm/lens-vm-go-host/types"
	stypes "github.com/lens-vm/lens-vm-go-sdk/types"

	"github.com/wasmerio/wasmer-go/wasmer"
)

var (
	DefaultOptions = &Options{
		Resolvers: []resolvers.Resolver{
			file.FileResolver{},
		},
	}
)

type Module struct {
	vm *VM

	id         string
	definition types.ResolvedModule

	dependancies map[string]*Module
	exportArgs   map[string]*json.RawMessage
	// lenses       map[string]*Module

	importObject *wasmer.ImportObject
	wmod         *wasmer.Module
	winst        *wasmer.Instance

	initialized bool
}

// setModuleImport sets the import entire module on the global VM scope
func (mod *Module) setModuleImport(name string, target *Module) {
	mod.vm.setModuleImport(name, target)
}

// setLensImport sets the individual lens functions on the module scope
func (mod *Module) setLensImport(name string, target *Module) {
	mod.dependancies[name] = target
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

type importSetter interface {
	setModuleImport(name string, mod *Module)
	setLensImport(name string, mod *Module)
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
	wasiEnv *wasmer.WasiEnvironment

	resolvers map[string]resolvers.Resolver

	resolverCtx context.Context
	execCtx     context.Context

	dgraph *dependancyGraph

	buffers map[stypes.BufferType][]byte

	initialized bool
}

func NewVM(opt *Options) *VM {
	if opt == nil {
		opt = DefaultOptions
	}
	wengine := wasmer.NewEngine()
	wstore := wasmer.NewStore(wengine)
	env, err := wasmer.NewWasiStateBuilder("lensvm-go-host").Finalize()
	if err != nil {
		panic(err)
	}
	vm := &VM{
		wengine:       wengine,
		wstore:        wstore,
		wasiEnv:       env,
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
	err := vm.makeDependancyGraph()
	var root string
	// grab any import path from our lens file
	// note: range over a map has an undefined
	// order, so effectively we get a random
	// element from the map here.
	for _, mod := range vm.lensImports {
		root = mod.id
		break
	}
	deps, err := vm.dgraph.SortOrder(root)
	if err != nil {
		return err
	}

	for _, dep := range deps {
		mod, ok := vm.moduleImports[dep]
		if !ok {
			return fmt.Errorf("Missing module dependancy: %s", dep)
		}
		if err := vm.moduleInit(mod); err != nil {
			return err
		}
	}

	return nil
}

// moduleInit initalizes the WASM module, including wiring all the
// depedant imports from both the VM host functions, and the dependancy
// module functions.
func (vm *VM) moduleInit(mod *Module) error {
	importObj, err := vm.wasiEnv.GenerateImportObject(vm.wstore, mod.wmod)
	if err != nil {
		return nil
	}

	mod.importObject = importObj
	if err := mod.RegisterFunc("env", "lensvm_get_buffer", vm.lensVMGetBufferBytes); err != nil {
		return nil
	}
	if err := mod.RegisterFunc("env", "lensvm_set_buffer", vm.lensVMSetBufferBytes); err != nil {
		return nil
	}

	// loop through the dependencies, and wire the exports/imports
	for lens, m := range mod.dependancies {
		// get the export from the dependancy
		fnName := formatExecName(lens)
		fn, err := m.winst.Exports.Get(fnName)
		if err != nil {
			return err
		}

		// add it to the importobject of the current module
		mod.importObject.Register("env", map[string]wasmer.IntoExtern{
			fnName: fn,
		})
	}

	// create new wasm instance
	inst, err := wasmer.NewInstance(mod.wmod, mod.importObject)
	if err != nil {
		return err
	}
	mod.winst = inst

	return nil
}

func formatExecName(name string) string {
	return fmt.Sprintf("lensvm_%s_exec", name)
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

		if _, err, _ := vm.addGlobalImport(name, resolvedMod); err != nil {
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
func (vm *VM) ImportModule(path string) (*Module, error) {
	rmod, err := vm.ResolveModule(path)
	if err != nil {
		return nil, err
	}

	mod, err, _ := vm.addGlobalImport("*", rmod)
	return mod, err
}

// ImportModuleFunction will resolve the module from the
// given path, and will import only the named module.
// If the Module file contains an array of defined modules
// it will scan all of them until it finds a match. If
// the module only defines a single module definition, it
// will import it if theres a name match.
func (vm *VM) ImportModuleFunction(name, path string) (*Module, error) {
	// if we're importing all the module functions, use the correct
	// function
	if name == "*" {
		return vm.ImportModule(path)
	}

	rmod, err := vm.ResolveModule(path)
	if err != nil {
		return nil, err
	}

	mod, err, _ := vm.addGlobalImport(name, rmod)
	return mod, err
}

// func (vm) ResolveContext()

// addGlobalImport adds the refenced lens function from the given ResolvedModule
// to the global (vm) scope.
func (vm *VM) addGlobalImport(name string, rmod types.ResolvedModule) (*Module, error, bool) {
	return vm.addScopedImport(vm, name, rmod)
}

// addScopedImport adds the referenced lens function from the given ResolvedModule
// to the supplied scope, can be either global (vm) or local (module).
// Recursively add all the necessary depenencies.
func (vm *VM) addScopedImport(scope importSetter, name string, rmod types.ResolvedModule) (*Module, error, bool) {
	if len(name) == 0 {
		return nil, errors.New("Missing name of import function"), false
	}
	if len(rmod.ID) == 0 {
		fmt.Println(rmod)
		return nil, errors.New("Invalid resolved module object. Missing ID"), false
	}
	if mod, exists := vm.moduleImports[rmod.ID]; exists {
		if name == "*" {
			for _, export := range rmod.Exports {
				scope.setLensImport(export.Name, mod)
			}
		} else {
			if moduleHasLensFunc(rmod, name) {
				scope.setLensImport(name, mod)
				return mod, nil, false
			}
			return nil, fmt.Errorf("Lens function '%s' is missing from module", name), false
		}
	}

	mod, err := vm.newModule(rmod)
	if err != nil {
		return nil, err, false
	}

	scope.setModuleImport(mod.id, mod)
	scope.setLensImport(name, mod)

	// loop and add all the modules' dependencies on this scope
	// recursively
	for k, v := range rmod.Imports {
		// check if empty
		if reflect.DeepEqual(types.ResolvedModule{}, v.Module) {
			continue
		}
		if _, err, _ := vm.addScopedImport(mod, k, v.Module); err != nil {
			return mod, err, false
		}
	}
	return mod, nil, true
}

// func (vm *VM) HasImport

// func (vm *VM) AddResolvedModule(rmod types.ResolvedModule) error {

// }

func (vm *VM) newModule(rmod types.ResolvedModule) (*Module, error) {
	// check ID and PackageBytes
	if rmod.ID == "" {
		return nil, errors.New("Invalid resolved module object. Missing ID")
	}
	if len(rmod.PackageBytes) == 0 {
		return nil, errors.New("Missing module wasm bytes")
	}
	exports := make(map[string]*json.RawMessage)
	for _, e := range rmod.Exports {
		exports[e.Name] = e.Arguments
	}

	wmod, err := wasmer.NewModule(vm.wstore, rmod.PackageBytes)
	if err != nil {
		return nil, err
	}
	return &Module{
		vm:           vm,
		id:           rmod.ID,
		definition:   rmod,
		dependancies: make(map[string]*Module),
		exportArgs:   exports,
		wmod:         wmod,
	}, nil
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
	if len(modFile.Exports) == 0 {
		return types.ResolvedModule{}, fmt.Errorf("Resolved module at path %s does not export any lens functions", path), false
	}

	// flatten the original modFile into a array.
	// It either contains the original modFile.Modules array
	// or an array of length 1, which is the root module in the
	// modFile
	root := types.ModuleToResolvedModule(modFile)
	root.ID = path

	for n, p := range modFile.Import {
		// impModule := types.ImportedModule{Path: p}
		mod, err, _ := vm.resolveModule(ctx, foundModules, p)
		if err != nil {
			return types.ResolvedModule{}, err, false
		}
		root.Imports[n] = types.ImportedModule{
			Path:   p,
			Module: mod,
		}
	}

	wasmBytes, err := vm.resolve(ctx, modFile.Package)
	if err != nil {
		return types.ResolvedModule{}, err, false
	}
	root.PackageBytes = wasmBytes
	root.ID = path

	return root, nil, true
	// return types.ResolvedModule{
	// 	ID:      path,
	// 	Modules: rMods,
	// }, nil, true
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

func (vm *VM) setModuleImport(name string, target *Module) {
	vm.moduleImports[name] = target
}

func (vm *VM) setLensImport(name string, target *Module) {
	vm.lensImports[name] = target
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

	for _, exp := range rmod.Exports {
		if name == exp.Name {
			return true
		}
	}

	return false
}

func isEmptyResolvedModule(mod types.ResolvedModule) bool {
	return (mod.Name == "" && len(mod.Exports) == 0)
}
