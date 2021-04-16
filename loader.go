package lensvm

import (
	"encoding/json"
	"errors"

	"github.com/lens-vm/lens-vm-go-host/resolvers"
	"github.com/lens-vm/lens-vm-go-host/types"
)

type LensLoader interface {
	Load() (types.LensFile, error)
	Path() string
}

type ModuleLoader interface {
	Load() (types.ModuleFile, error)
	Path() string
}

type genericLoader struct {
	resolver resolvers.Resolver
	path     string
}

func (l genericLoader) Path() string {
	return l.path
}

func (l genericLoader) resolve() ([]byte, error) {
	if l.path == "" {
		return nil, errors.New("LensLoader path is empty")
	}
	return l.resolver.Resolve(l.path)
}

type lensFileLoader struct {
	genericLoader
}

func LensFileLoader(path string) LensLoader {
	return lensFileLoader{
		genericLoader{
			path: path,
		},
	}
}

func (l lensFileLoader) Load() (types.LensFile, error) {
	var lf types.LensFile
	buf, err := l.resolve()
	if err != nil {
		return lf, err
	}

	err = json.Unmarshal(buf, &lf)
	return lf, err
}

type moduleFileLoader struct {
	genericLoader
}

func ModuleFileLoader(path string) ModuleLoader {
	return moduleFileLoader{
		genericLoader{
			path: path,
		},
	}
}

func (l moduleFileLoader) Load() (types.ModuleFile, error) {
	var mf types.ModuleFile
	buf, err := l.resolve()
	if err != nil {
		return mf, err
	}

	err = json.Unmarshal(buf, &mf)
	return mf, err
}
