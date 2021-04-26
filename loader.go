package lensvm

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/lens-vm/lens-vm-go-host/resolvers"
	"github.com/lens-vm/lens-vm-go-host/resolvers/file"
	"github.com/lens-vm/lens-vm-go-host/types"
)

type LensLoader interface {
	Load(context.Context) (types.LensFile, error)
	Path() string
}

type ModuleLoader interface {
	Load(context.Context) (types.ModuleFile, error)
	Path() string
}

type genericLoader struct {
	resolver resolvers.Resolver
	path     string
}

func (l genericLoader) Path() string {
	return l.path
}

func (l genericLoader) resolve(ctx context.Context) ([]byte, error) {
	if l.path == "" {
		return nil, errors.New("LensLoader path is empty")
	}
	return l.resolver.Resolve(ctx, l.path)
}

type lensFileLoader struct {
	genericLoader
}

func LensFileLoader(path string) LensLoader {
	return lensFileLoader{
		genericLoader{
			resolver: file.FileResolver{},
			path:     path,
		},
	}
}

func (l lensFileLoader) Load(ctx context.Context) (types.LensFile, error) {
	var lf types.LensFile
	buf, err := l.resolve(ctx)
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

func (l moduleFileLoader) Load(ctx context.Context) (types.ModuleFile, error) {
	var mf types.ModuleFile
	buf, err := l.resolve(ctx)
	if err != nil {
		return mf, err
	}

	err = json.Unmarshal(buf, &mf)
	return mf, err
}
