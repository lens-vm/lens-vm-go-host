package file

import (
	"context"
	"io/ioutil"
)

type FileResolver struct{}

func (f FileResolver) Scheme() string {
	return "file"
}

func (f FileResolver) Resolve(ctx context.Context, path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}
