package main

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

func TestNone(t *testing.T) {
	logger.Info("No tests currently implemented")
}
