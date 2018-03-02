package devtools

import (
	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks

	"github.com/karlmutch/semver" // Forked copy of https://github.com/Masterminds/semver

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
)

type GitInfo struct {
	Dir    string       // The directory that is being used as the repository root directory
	Branch string       // The current branch the repo is checkedout against
	Err    errors.Error // If initialization resulted in an error it may have been stored in this variable`
}

type MetaData struct {
	SemVer *semver.Version
	Git    *GitInfo
}
