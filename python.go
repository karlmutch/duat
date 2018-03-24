// +build ignore

package main

// Compiled with export GODEBUG=cgocheck=0
// go build -buildmode=c-shared -o gen/test.so python.go

import (
	"C"
	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

//export Check
func Check() (err string) {
	return errors.New("custom error").With("stack", stack.Trace().TrimRuntime()).Error()
}
