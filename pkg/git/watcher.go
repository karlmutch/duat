package duatgit

// This file contains the implementation of serveral functions that are
// useful for monitoring git repositories.  This is useful for when
// CI/CD pipelines are unable to establish hook servers to monitor traffic
// from github and the like.

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/jjeffery/kv"
	"github.com/karlmutch/base62"
	"github.com/karlmutch/deepcopier"
	"github.com/karlmutch/stack"

	"gopkg.in/src-d/go-git.v4" // Not forked due to depency tree being too complex, src-d however are a serious org so I dont expect the repo to disappear
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

var (
	interval = time.Duration(45 * time.Second)

	initWatcher sync.Once
)

func getDirHash(repoURL string) (encodedHash string) {
	h := sha256.New()
	h.Write([]byte(repoURL))
	i := new(big.Int)
	if _, isOK := i.SetString(fmt.Sprintf("%x", h.Sum(nil)), 16); !isOK {
		return ""
	}
	return base62.EncodeBigInt(i)
}

func IsDirEmpty(name string) (empty bool, errGo error) {
	f, errGo := os.Open(name)
	if errGo != nil {
		return false, errGo
	}
	defer f.Close()

	// Only needed that we read a single entry
	_, errGo = f.Readdir(1)

	// If EOF we have an empty directory
	return errGo == io.EOF, errGo
}

type LoggerSink struct {
	Msg  string
	Args []interface{}
}

func reportError(errGo error, loggerC chan<- *LoggerSink) {
	if errGo != nil {
		if loggerC != nil {
			select {
			case loggerC <- &LoggerSink{
				Msg: errGo.Error(),
				Args: []interface{}{
					"stack",
					stack.Trace().TrimRuntime(),
				},
			}:
				return
			case <-time.After(time.Second):
			}
		}
		fmt.Println("error sending", errGo.Error(), stack.Trace().TrimRuntime())
	}
}

func (gw *GitWatcher) watcher(ctx context.Context, interval time.Duration, loggerC chan<- *LoggerSink) {

	defer close(gw.Stopped)

	// On the first pass only we check almost immediately
	firstPass := true
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Copy the list of repositories that are to be checked
			gw.Lock()
			checkRepos := make(map[string]*GitOptions, len(gw.Repos))
			for k, v := range gw.Repos {
				copied := &GitOptions{}
				deepcopier.Copy(v).To(copied)
				checkRepos[k] = copied
			}
			gw.Unlock()
			for k, v := range checkRepos {
				dirName := filepath.Join(gw.Dir, k)
				if _, errGo := os.Stat(dirName); os.IsNotExist(errGo) {
					// Git clone into this name
					if _, errGo = git.PlainCloneContext(ctx, dirName, false, v.CloneOptions); errGo != nil {
						reportError(errGo, loggerC)
						continue
					}
				}

				// Opens a cloned repository
				repo, errGo := git.PlainOpen(dirName)
				if errGo != nil {
					reportError(errGo, loggerC)
					continue
				}
				if len(v.Branch) != 0 {
					v.Branch = "master"
				}

				tree, errGo := repo.Worktree()
				if errGo != nil {
					reportError(errGo, loggerC)
					continue
				}

				ref := path.Join("refs", "heads", v.Branch)
				errGo = tree.Checkout(
					&git.CheckoutOptions{
						Branch: plumbing.ReferenceName(ref),
						Create: false,
						Force:  true,
					})
				if errGo != nil {
					reportError(errGo, loggerC)
					continue
				}

				errGo = tree.Pull(
					&git.PullOptions{
						ReferenceName: plumbing.ReferenceName(path.Join("refs", "heads", v.Branch)),
					})
				if errGo != nil && errGo != git.NoErrAlreadyUpToDate {
					reportError(errGo, loggerC)
					continue
				}

				head, errGo := repo.Head()
				if errGo != nil {
					reportError(errGo, loggerC)
					continue
				}

				refHash := head.Hash().String()
				lastKnownHash := ""

				update := false

				// Check for updates on any of the repositories by looking at the
				// URL hash file inside the main working directory manifest file
				manifestFN := filepath.Join(gw.Dir, k+".last")
				if _, errGo := os.Stat(manifestFN); os.IsNotExist(errGo) {
					update = true
				} else {
					if content, errGo := ioutil.ReadFile(manifestFN); errGo != nil || len(content) != 0 {
						update = true
					} else {
						lastKnownHash = string(content)
					}
				}
				if lastKnownHash != refHash {
					update = true
				}

				if update {
					select {
					case loggerC <- &LoggerSink{
						Msg: fmt.Sprintf("updating to %v", refHash),
						Args: []interface{}{
							"stack",
							stack.Trace().TrimRuntime(),
						},
					}:
					case <-time.After(time.Second):
					}
					if errGo = ioutil.WriteFile(manifestFN, []byte(refHash), 0600); errGo != nil {
						reportError(errGo, loggerC)
						continue
					}
				}
			}

		case <-ctx.Done():
			return
		}
		// The first pass will first the check immediately, subsequent passes will wait
		if firstPass {
			firstPass = false
			ticker.Stop()
			ticker = time.NewTicker(interval)

		}
	}
}

type GitOptions struct {
	CloneOptions *git.CloneOptions
	Branch       string
}

type GitWatcher struct {
	Dir     string
	Repos   map[string]*GitOptions
	Remove  bool
	Ctx     context.Context
	Cancel  context.CancelFunc
	Stopped chan struct{}
	sync.Mutex
}

func NewGitWatcher(ctx context.Context, baseDir string, loggerC chan<- *LoggerSink) (watcher *GitWatcher, err kv.Error) {

	watcher = &GitWatcher{
		Dir:     baseDir,
		Repos:   map[string]*GitOptions{},
		Remove:  false,
		Stopped: make(chan struct{}, 1),
	}

	if len(baseDir) == 0 {
		tmp, errGo := ioutil.TempDir("", "git-watcher")
		if errGo != nil {
			return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
		watcher.Dir = tmp
		watcher.Remove = true
	}

	defer func() {
		initWatcher.Do(
			func() {
				go watcher.watcher(ctx, time.Duration(34*time.Second), loggerC)
			})
	}()

	return watcher, nil
}

func (gw *GitWatcher) Add(url string, branch string, token string) (err kv.Error) {
	gitOptions := &GitOptions{
		CloneOptions: &git.CloneOptions{
			URL:               url,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		},
		Branch: branch,
	}

	if len(token) != 0 {
		// The intended use of a GitHub personal access token is in replace of your password
		// because access tokens can easily be revoked.
		// https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/
		gitOptions.CloneOptions.Auth = &http.BasicAuth{
			Username: "duat", // yes, this can be anything except an empty string
			Password: token,
		}
	}

	urlHash := getDirHash(url)

	gw.Lock()
	defer gw.Unlock()

	if _, isPresent := gw.Repos[urlHash]; isPresent {
		return kv.NewError("url already present in the git watcher").With("url", url, "hash", urlHash, "stack", stack.Trace().TrimRuntime())

	}

	gw.Repos[urlHash] = gitOptions

	return nil
}

func (gw *GitWatcher) Stop(ctx context.Context) (orderly bool) {

	orderly = true

	gw.Lock()
	defer gw.Unlock()

	// Signal the desire that things be stopped
	gw.Cancel()

	// Wait for a second for an orderly shutdown and then continue
	// regardless
	select {
	case <-gw.Stopped:
	case <-ctx.Done():
		orderly = false
	}

	// If storage needs releasing we now do it.  This should
	// only happen if the storage area for the repository was
	// known and supplied by the caller
	//
	if gw.Remove {
		os.RemoveAll(gw.Dir)
	}

	return orderly
}
