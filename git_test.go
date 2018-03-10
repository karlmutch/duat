package duat

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack

	"gopkg.in/src-d/go-git.v4" // Not forked due to depency tree being too complex, src-d however are a serious org so I dont expect the repo to disappear
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func TestGitLoad(t *testing.T) {
	// Get ourselves a working git repo
	baseDir, errGo := ioutil.TempDir("", "test-git-load")
	if errGo != nil {
		t.Error(errors.Wrap(errGo, "unable to create test data directory").With("stack", stack.Trace().TrimRuntime()))
		return
	}

	defer func() {
		if !t.Failed() {
			os.RemoveAll(baseDir) // clean up
		}
	}()

	gitURL := "https://github.com/karlmutch/no-code"
	_, errGo = git.PlainClone(baseDir, false, &git.CloneOptions{
		URL: gitURL,
	})
	if errGo != nil {
		t.Error(errors.Wrap(errGo, "unable to git clone the test repository").With("stack", stack.Trace().TrimRuntime()).With("url", gitURL))
		return
	}

	md := &MetaData{}
	err := md.LoadGit(baseDir, true)
	if err != nil {
		msg := "unable to locate git clone in the git directory with scanParents on"
		t.Error(errors.Wrap(err, msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", baseDir))
		return
	}
	if md.Git.Dir != baseDir {
		msg := "located git clone testDir in the git directory did not match expected dir with scanParents on"
		t.Error(errors.Wrap(err, msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", baseDir).With("testDir", md.Git.Dir))
		return
	}
	if md.Git.URL != gitURL {
		msg := "located git clone URL did not match the requested url"
		t.Error(errors.Wrap(err, msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", baseDir).With("testDir", md.Git.Dir).With("extractedURL", md.Git.URL))
		return
	}

	w, errGo := md.Git.Repo.Worktree()
	if errGo != nil {
		t.Error(errors.Wrap(errGo, "unable to prepare the work tree for qit").With("stack", stack.Trace().TrimRuntime()).With("url", gitURL))
		return
	}

	descend := []string{"1", "2"}
	dir := baseDir
	for _, nextDir := range descend {
		dir = filepath.Join(dir, nextDir)

		// Make sure it exists before doing any fancy git checking and navigation

		md.Git = nil
		err = md.LoadGit(dir, false)
		if err == nil {
			msg := "located git clone in a git subdirectory with scanParents off"
			t.Error(errors.Wrap(err, msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", dir))
			return
		}
		md.Git = nil
		err = md.LoadGit(dir, true)
		if err != nil {
			msg := "unable to locate git clone in a git subdirectory with scanParents on"
			t.Error(errors.Wrap(err, msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", dir))
			return
		}
		if md.Git.Dir != baseDir {
			msg := "located git clone testDir in a git subdirectory did not match expected dir with scanParents off"
			t.Error(errors.New(msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", baseDir).With("testDir", md.Git.Dir))
			return
		}
		if md.Git.Branch != "master" {
			msg := "located git clone testDir in a git subdirectory did not appear to be on branch master"
			t.Error(errors.New(msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", baseDir).With("testDir", md.Git.Dir))
			return
		}
	}
	// Find a known tag for testing and then test that we can nvigate and then identify it from the
	// LoadGit method

	tags, _ := md.Git.Repo.Tags()
	_ = tags.ForEach(func(t *plumbing.Reference) error {
		if t.Name().String() == "refs/tags/first-tag" {
			errGo = w.Checkout(&git.CheckoutOptions{
				Hash: t.Hash(),
			})
		}
		return nil
	})

	md.Git = nil
	err = md.LoadGit(baseDir, false)
	if err != nil {
		msg := "unable to locate git clone in the git directory with scanParents off"
		t.Error(errors.Wrap(err, msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", baseDir))
		return
	}
	if md.Git.Dir != baseDir {
		msg := "located git clone testDir in the git directory did not match expected dir with scanParents off"
		t.Error(errors.New(msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", baseDir).With("testDir", md.Git.Dir))
		return
	}
	if md.Git.Tag != "first-tag" {
		msg := "located git clone with a checkout of a tag could not located the same tag using commit hashes"
		t.Error(errors.New(msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", baseDir).With("testDir", md.Git.Dir))
		return
	}
	if md.Git.Branch != "HEAD" {
		msg := "located git clone testDir in a git subdirectory did not appear to be properly detached"
		t.Error(errors.New(msg).With("stack", stack.Trace().TrimRuntime()).With("url", gitURL).With("dir", baseDir).With("testDir", md.Git.Dir))
		return
	}
}
