package file

import (
	"context"
	"io/ioutil"
	"strings"
)

type FileResolver struct{}

func (f FileResolver) Scheme() string {
	return "file"
}

func (f FileResolver) Resolve(ctx context.Context, path string) ([]byte, error) {
	if strings.Contains(path, f.Scheme()+"://") {
		path = strings.Replace(path, f.Scheme()+"://", "", -1)
	}
	return ioutil.ReadFile(path)
}
