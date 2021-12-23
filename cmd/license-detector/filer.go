package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/go-enry/go-license-detector/v4/licensedb/filer"
)

// Implementation of the filer for license scanning to allow explicit file lists
type localFiler struct {
	files []string
}

// FromFiles returns a Filer that allows accessing over a list of explicit files
func FromFiles(files []string) (aFiler *localFiler, errGo error) {
	aFiler = &localFiler{
		files: make([]string, 0, len(files)),
	}

	for _, aFile := range files {
		fi, errGo := os.Stat(aFile)
		if errGo != nil {
			return nil, errors.Wrapf(errGo, "cannot create Filer from "+aFile)
		}
		if fi.IsDir() {
			return nil, errors.New("a directory was specified, only file names are permitted " + aFile)
		}

		fullPath, errGo := filepath.Abs(aFile)
		if errGo != nil {
			return nil, errors.Wrapf(errGo, "cannot resolve file "+aFile)
		}
		aFiler.files = append(aFiler.files, fullPath)
	}
	return aFiler, nil
}

func (*localFiler) ReadFile(path string) (buffer []byte, errGo error) {
	return ioutil.ReadFile(path)
}

func (aFiler *localFiler) ReadDir(path string) (files []filer.File, errGo error) {
	files = make([]filer.File, 0, len(aFiler.files))
	for _, file := range aFiler.files {
		files = append(files, filer.File{
			Name:  file,
			IsDir: false,
		})
	}
	return files, nil
}

func (aFiler *localFiler) Close() {}

func (aFiler *localFiler) PathsAreAlwaysSlash() bool {
	return false
}
