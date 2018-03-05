package duat

import (
	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks

	"github.com/karlmutch/semver" // Forked copy of https://github.com/Masterminds/semver

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors

	docker "github.com/docker/docker/client"
	"gopkg.in/src-d/go-git.v4" // Not forked due to depency tree being too complex, src-d however are a serious org so I dont expect the repo to disappear
)

type GitInfo struct {
	URL    string          // The URL of the main git remote being used
	Repo   *git.Repository // A handle to the srd-d git object
	Dir    string          // The directory that is being used as the repository root directory
	Branch string          // The current branch the repo is checkedout against
	Tag    string          // The tag for the current commit if present
	Err    errors.Error    // If initialization resulted in an error it may have been stored in this variable`
}

type MetaData struct {
	Dockers map[string]docker.Client
	SemVer  *semver.Version
	Git     *GitInfo
}
