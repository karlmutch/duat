// +build ignore

package main

// Compiled with export GODEBUG=cgocheck=0
// go build -buildmode=c-shared -o gen/test.so python.go

import (
	"C"

	"github.com/jjeffery/kv"     // Forked copy of https://github.com/jjeffery/kv
	"github.com/karlmutch/stack" // Forked copy of https://github.com/go-stack/stack
)

//export Check
func Check() (err string) {
	return kv.NewError("custom error").With("stack", stack.Trace().TrimRuntime()).Error()
}
