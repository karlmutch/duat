package duat

import (
	"flag"
	"os"
	"testing"

	"github.com/karlmutch/envflag"
)

func TestMain(m *testing.M) {
	if !flag.Parsed() {
		envflag.Parse()
	}

	if resultCode := m.Run(); resultCode != 0 {
		os.Exit(resultCode)
	}
}
