// Copyright (c) 2021 The duat Authors. All rights reserved.  Issued under the MIT license.
package duat

import (
	"os"
	"testing"

	"github.com/go-test/deep"
)

func TestGithub(t *testing.T) {
	md, err := NewMetaData(".", "README.md")
	if err != nil {
		t.Fatal(err)
	}
	token := os.Getenv("GITHUB_TOKEN")
	if len(token) == 0 {
		t.Skip("github token unavailable")
	}

	released, err := md.HasReleased(token, "0.15.2", []string{"github-release-linux-amd64", "bogus"})
	if err != nil {
		t.Fatal(err)
	}
	if diff := deep.Equal(released, []string{"github-release-linux-amd64"}); diff != nil {
		t.Fatal(diff)
	}
}
