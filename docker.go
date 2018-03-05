package duat

import (
	"strings"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
)

// ScrubRepository is used to transform module names into docker image name compliant strings.
// The definition of a repository name string can be found at https://github.com/moby/moby/blob/master/image/spec/v1.2.md
//
func (*MetaData) ScrubDockerRepo(name string) (repoName string, err errors.Error) {
	for _, aChar := range name {
		if aChar < '0' || aChar > 'z' || (aChar > '9' && aChar < 'A') || (aChar > 'Z' && aChar < 'a') {
			if aChar != '.' && aChar != '_' && aChar != '-' {
				repoName += "-"
				continue
			}
		}
		repoName += string(aChar)
	}

	return strings.ToLower(repoName), nil
}
