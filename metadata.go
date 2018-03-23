package duat

import (
	"net/url"
	"os"
	"path/filepath"

	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks
	"github.com/karlmutch/semver" // Forked copy of https://github.com/Masterminds/semver

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack

	docker "github.com/docker/docker/client"

	"gopkg.in/src-d/go-git.v4" // Not forked due to depency tree being too complex, src-d however are a serious org so I dont expect the repo to disappear
)

type GitInfo struct {
	URL    url.URL         // The URL of the main git remote being used
	Repo   *git.Repository // A handle to the srd-d git object
	Dir    string          // The directory that is being used as the repository root directory
	Branch string          // The current branch the repo is checkedout against
	Tag    string          // The tag for the current commit if present
	Hash   string          // The hash for the current commit
	Token  string          // If the GITHUB token was available then it will be saved here
	Err    errors.Error    // If initialization resulted in an error it may have been stored in this variable`
}

type MetaData struct {
	Dockers map[string]docker.Client
	SemVer  *semver.Version
	Module  string // A string name for the software component that is being handled
	VerFile string // The file that is being used as the reference for version data
	Git     *GitInfo
}

func (md *MetaData) Clear() {
	md.Dockers = map[string]docker.Client{}
	md.SemVer = nil
	md.Module = ""
	md.VerFile = ""
	md.Git = nil
}

// NewMetaData will switch to the indicated project directory and will load
// appropriate project information into the meta-data structure returned to
// the caller
//
func NewMetaData(dir string, verFile string) (md *MetaData, err errors.Error) {

	md = &MetaData{}

	cwd, errGo := os.Getwd()
	if errGo != nil {
		return nil, errors.Wrap(errGo, "current directory unknown").With("stack", stack.Trace().TrimRuntime())
	}
	md.Module = filepath.Base(cwd)

	if len(dir) > 0 && dir != "." {
		dirCtx := ""
		if dir[0] == '/' {
			md.Module = filepath.Base(dir)
			dirCtx = dir
		} else {
			dirCtx = filepath.Join(cwd, dir)
			md.Module = filepath.Base(dirCtx)
		}
		path, errGo := filepath.Abs(dirCtx)
		if errGo != nil {
			return nil, errors.Wrap(errGo, "could not resolve the project directory").With("dir", dirCtx).With("stack", stack.Trace().TrimRuntime())
		}

		if cwd != path {
			if errGo := os.Chdir(path); errGo != nil {
				return nil, errors.Wrap(errGo, "could not change to the project directory").With("dir", path).With("stack", stack.Trace().TrimRuntime())
			}
		}
	}

	if err = md.LoadGit(cwd, true); err != nil {
		return nil, err
	}

	// The main README.md will be at the git repos top directory
	md.VerFile = filepath.Join(md.Git.Dir, verFile)
	if _, err = md.LoadVer(md.VerFile); err != nil {
		return nil, err
	}

	// Ensure that the module name is made docker compatible
	if md.Module, err = md.ScrubDockerRepo(md.Module); err != nil {
		return nil, err
	}
	return md, nil
}
