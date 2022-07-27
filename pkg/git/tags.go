package git

import (
	"os"
	"sort"
	"strings"

	"github.com/go-stack/stack"
	"github.com/jjeffery/kv"

	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks

	"github.com/Masterminds/semver"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/goreleaser/chglog"
	// Forked copy of https://github.com/GoBike/envflag
	// Using a forked copy of this package results in build issues
)

type TagDetails struct {
	Tag       *semver.Version
	hashStart plumbing.Hash
	hashEnd   plumbing.Hash
}

type tagHistory struct {
	BaseDir string
	Tags    []*TagDetails
}

func TagHistory() (history *tagHistory, err kv.Error) {

	history = &tagHistory{
		Tags: []*TagDetails{},
	}

	wd, errGo := os.Getwd()
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	// Examine the directory heirarchy looking for a git base dir
	repo, errGo := chglog.GitRepo(wd, true)
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	// Save the git root directory name
	wt, errGo := repo.Worktree()
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	history.BaseDir = wt.Filesystem.Root()

	// Get the first commit
	firstCommit := plumbing.ZeroHash
	commitIter, errGo := repo.CommitObjects()
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	errGo = commitIter.ForEach(func(commit *object.Commit) error {
		if len(commit.ParentHashes) == 0 {
			firstCommit = commit.Hash
		}
		return nil
	})
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	if firstCommit == plumbing.ZeroHash {
		return nil, kv.NewError("first commit not found")
	}

	iter, errGo := repo.Tags()
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	// Have collection of semantic versions that will be sorted using semver semantics
	sortedTags := []*semver.Version{}
	tagPrefix := "refs/tags/"

	// Contains a map of tag informated indexed using the semantic version string
	tags := map[string]*TagDetails{}

	// Get tags and look for semver compliant tags, saving hash information as we go
	if errGo := iter.ForEach(func(ref *plumbing.Reference) error {
		tag := ref.Name().String()
		if strings.HasPrefix(tag, tagPrefix) {
			tagParts := strings.SplitAfter(tag, tagPrefix)
			ver, errGo := semver.NewVersion(tagParts[1])
			if errGo != nil {
				return nil
			}
			tags[ver.Original()] = &TagDetails{Tag: ver, hashEnd: ref.Hash()}
			sortedTags = append(sortedTags, ver)
		}
		return nil
	}); errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	// Sort semver tags collection, in promotion order
	sort.Sort(semver.Collection(sortedTags))

	// Go backthrough our known tags and get the points at which each of the tagged versions development began
	// and update the tags with their first commit ID
	hashStart := firstCommit
	for _, tag := range sortedTags {
		tags[tag.Original()].hashStart = hashStart
		hashStart = tags[tag.Original()].hashEnd
		// Store the tags in semver collation order
		history.Tags = append(history.Tags, tags[tag.Original()])
	}

	return history, nil
}
