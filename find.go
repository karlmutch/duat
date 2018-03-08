package duat

//. This file contains the implementation of functions related to common shell style operations such
// as finding files etc

import (
	"path/filepath"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

func (*MetaData) FileFind(pattern string) (matches []string, err errors.Error) {
	// Matches will be produced in lexographical order, see comments in
	// https://golang.org/src/path/filepath/match.go?s=5609:5664#L224
	matches, errGo := filepath.Glob(pattern)
	if errGo != nil {
		return nil, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	return matches, nil
}
