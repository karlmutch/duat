package duat

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/jjeffery/kv"
	"github.com/karlmutch/semver"
)

// ScrubeForDocker is used to transform module names into docker image name compliant strings.
// The definition of a repository name string can be found at https://github.com/moby/moby/blob/master/image/spec/v1.2.md
//
func (*MetaData) ScrubForDocker(name string) (repoName string, err kv.Error) {
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

func (md *MetaData) generateImageName(semVer *semver.Version) (repo string, version string, prerelease bool, err kv.Error) {
	// Get the git repo name and the parent which will have been used to name the
	// containers being created during our build process
	gitParts := strings.Split(md.Git.URL.EscapedPath(), "/")

	label := gitParts[len(gitParts)-1]
	if strings.HasSuffix(label, ".git") {
		label = strings.TrimSuffix(label, ".git")
	}

	// Look for pre-release components within the version string
	preParts := strings.Split(semVer.Prerelease(), "-")

	repoName := fmt.Sprintf("%s/%s/%s", gitParts[len(gitParts)-2], label, md.Module)
	// docker repositories only use lowercase _ and -
	dockerRepo := strings.Builder{}
	for i, char := range repoName {
		if unicode.IsUpper(char) && i != 0 {
			dockerRepo.WriteRune('-')
		}
		dockerRepo.WriteRune(unicode.ToLower(char))
	}
	return dockerRepo.String(), semVer.String(), len(preParts) >= 2, nil
}

// caller should
// Clean the semver and save the new one after the push

func (md *MetaData) GenerateImageName() (repo string, version string, prerelease bool, err kv.Error) {
	return md.generateImageName(md.SemVer)
}
