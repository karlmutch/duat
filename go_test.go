package duat

import (
	"testing"

	"github.com/go-test/deep"
)

// This file contains a number of test functions for the go compilation, generator
// and test features of this package

// TestGoGenerator will use the assets directory to detect packages that contain
// go generate code
//
func TestGoGenerator(t *testing.T) {

	// Test assets
	possibles, err := FindGoGenerateDirs([]string{"assets"}, []string{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"assets/gogen/generate.go"}

	if diff := deep.Equal(expected, possibles); diff != nil {
		t.Error(diff)
	}
}
