package file

import (
	"io/ioutil"
)

type FileResolver struct{}

func (f FileResolver) Scheme() string {
	return "file"
}

func (f FileResolver) Resolve(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}
