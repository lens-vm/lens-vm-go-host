package types

import "encoding/json"

type ModuleFile struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Url         string           `json:"url,omitempty"`
	Arguments   *json.RawMessage `json:"arguments"`
	Import      ImportDefinition `json:"import"`
	Runtime     string           `json:"runtime"`
	Language    string           `json:"language"`
	Package     string           `json:"module"`

	// Modules contains a list of ModuleFileDefinitions
	// if this ModuleFile defines a collection of
	// lens modules, and not just one.
	// This is only used if the rest of the parameters are
	// empty to ensure that this definition file
	// is EITHER one module or a list of modules.
	Modules []ModuleFile `json:"modules"`
}

type LensFile struct {
	Import ImportDefinition              `json:"import"`
	Lenses []map[string]*json.RawMessage `json:"lenses"`
}

type ImportDefinition map[string]string

// type GenericDefinition map[string]*json.RawMessage

type ResolvedModule struct {
	ID           string
	Name         string
	Description  string
	Url          string
	Arguments    *json.RawMessage
	Import       map[string]ImportedModule
	Runtime      string
	Language     string
	PackagePath  string
	PackageBytes []byte

	Modules []ResolvedModule
}

type ImportedModule struct {
	Path   string
	Module ResolvedModule
}

// ModuleToReolvedModule does a basic syntax translation
// from a ModuleFile to a ResolvedModule type. It ignores
// the []Modules arrays in both
func ModuleToResolvedModule(f ModuleFile) ResolvedModule {
	return ResolvedModule{
		Name:        f.Name,
		Description: f.Description,
		Url:         f.Url,
		Arguments:   f.Arguments,
		Import:      make(map[string]ImportedModule),
		Runtime:     f.Runtime,
		Language:    f.Language,
		PackagePath: f.Package,
	}
}
