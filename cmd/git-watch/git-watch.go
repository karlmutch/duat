package main

// This file contains the main function for a git cimmit watcher that upon a git
// commit occuring will activate a kubernetes job based upon a template.  In this
// case this could be a Mikasu build job to allow a CI pipeline to be initiated.
//
// The rational behind triggering pipeline in this manner is that it provides a
// minimal viable way of self hosting a quay.io service.  It cannot match this
// service with things such as security scanning of resulting images triggered
// by cimmits but will perform at least minimal self hosted docker builds within
// a users self provisioned Kubernetes cluster.  Subsequent images from Mikasu
// can then be pushed to a vanilla docker image registry where scans and the like
// can be performed.
//
// This tool does not depend upon webhooks.  This tool will write the last known
// commit ID within a user specified directory to allow the changes to be tracked
// across restarts, for example persistent volumes in Kubernetes.
//
import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jjeffery/kv"
	"github.com/karlmutch/duat/pkg/git"
	"github.com/karlmutch/duat/pkg/kubernetes"
	"github.com/karlmutch/duat/version"

	"github.com/karlmutch/stack"
	colorable "github.com/mattn/go-colorable"

	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envflag

	"github.com/mgutz/logxi" // Using a forked copy of this package results in build issues

	"github.com/google/uuid"
)

var (
	logger = logxi.NewLogger(logxi.NewConcurrentWriter(colorable.NewColorableStderr()), "git-watch")

	githubToken = flag.String("github-token", "", "A github token that can be used to access the repositories that will be watched")
	verbose     = flag.Bool("v", false, "When enabled will print internal logging for this tool")

	triggerScript    = flag.String("script", "", "The name of a shell script file that will be run on a change being detected, env var GIT_HOME will be set to indicate the repo directory of the activated script")
	triggerJob       = flag.String("job-template", "", "The regular expression used to locate the Kubernetes job specification that is run on a change being detected, env var GIT_HOME will be set to indicate the repo directory of the activated script")
	triggerNamespace = flag.String("namespace", "", "Overrides the defaulted namespace for pods and other resources that are spawned by this command")

	gitRepos    = flag.String("urls", "", "One of more git repositories to monitor for changes")
	gitBranches = flag.String("branches", "", "A branch for each repository to needs watching, defaults to using 'master' for all repositories")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       Git Commit watcher and trigger (git-watch)      ", version.GitHash, "    ", version.BuildTime)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Environment Variables:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "options can also be extracted from environment variables by changing dashes '-' to underscores and using upper case.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "log levels are handled by the LOGXI env variables, these are documented at https://github.com/mgutz/logxi")
}

type JobTracker struct {
	jobs map[string]kubernetes.StartJob
	sync.Mutex
}

// extractMap is used to turn a list of interfaces with alternating keys and values as individual items
// into a map for cases where the items in the list could be dealth with as strings.  The use case for this
// function is in relation to the kv package with lists.
//
func extractMap(list []interface{}) (withs map[string]string) {
	withs = map[string]string{}
	for i, v := range list {
		if i%2 != 0 {
			continue
		}
		key, ok := v.(string)
		if !ok {
			continue
		}
		value, ok := list[i+1].(string)
		if !ok {
			continue
		}
		withs[key] = value
	}
	return withs
}

func main() {

	if !flag.Parsed() {
		envflag.Parse()
	}

	// Turn off logging regardless of the default levels if the verbose flag is not enabled.
	// By design this is a CLI tool and outputs information that is expected to be used by shell
	// scripts etc
	//
	if !*verbose {
		logger.SetLevel(logxi.LevelError)
	}

	logger.Debug(fmt.Sprintf("%s built at %s, against commit id %s\n", os.Args[0], version.BuildTime, version.GitHash))

	if len(flag.Args()) > 2 {
		usage()
		fmt.Fprintf(os.Stderr, "too many (%d - %v), arguments.\n", len(flag.Args()), flag.Args())
		os.Exit(-1)
	}

	stateDir := "/tmp/git-watcher"
	if len(flag.Args()) == 2 {
		stateDir = flag.Arg(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	reportC := make(chan *git.LoggerSink, 1)
	go func() {
		for {
			select {
			case report := <-reportC:
				logger.Warn(report.Msg, report.Args...)
			case <-ctx.Done():
				return
			}
		}
	}()

	watcher, err := git.NewGitWatcher(ctx, stateDir, reportC)
	if err != nil {
		logger.Info(err.Error())
	}

	repos := strings.Split(*gitRepos, ",")
	branches := strings.Split(*gitBranches, ",")

	// Check that we have at least one repository that is to be monitored
	if len(repos) == 0 {
		fmt.Fprintf(os.Stderr, "%v\n",
			kv.NewError("no github repositories were specified").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash))
		os.Exit(-1)
	}
	if len(branches) > len(repos) {
		fmt.Fprintf(os.Stderr, "%v\n",
			kv.NewError("more branches were specified than github repositories").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash))
		os.Exit(-1)
	}

	jobTriggerC := make(chan kubernetes.StartJob, 1)
	defer close(jobTriggerC)

	jobTracking := &JobTracker{
		jobs: map[string]kubernetes.StartJob{},
	}

	// Create a channel that receives notifications of repo changes, and also
	// the handler function that deals with the notifications
	trackingC := make(chan git.Change, 1)
	go func(ctx context.Context, triggerC chan git.Change) {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-triggerC:
				start := kubernetes.StartJob{
					ID:         uuid.New().String(),
					Dir:        msg.Dir,
					Dockerfile: "",
					Env:        map[string]string{},
				}

				switch *triggerNamespace {
				case "":
					start.Namespace = start.ID
				case "generated":
					start.Namespace = uuid.New().String()
				default:
					start.Namespace = *triggerNamespace
				}

				jobTracking.Lock()
				jobTracking.jobs[start.ID] = start
				jobTracking.Unlock()

				jobTriggerC <- start
			}
		}
	}(ctx, trackingC)

	doneC := make(chan *kubernetes.Status, 128)
	go func() {
		for {
			select {
			case msg := <-doneC:
				text, list := kv.Parse([]byte(msg.Msg.Error()))
				withs := extractMap(list)

				if msg.Level == logxi.LevelNotice && string(text) == "success" {
					jobTracking.Lock()
					job, isPresent := jobTracking.jobs[msg.ID]
					jobTracking.Unlock()
					if isPresent {
						logger.Info("job completed", "id", msg.ID, "dir", job.Dir, "namespace", withs["namespace"])
					} else {
						logger.Info("job completed", "id", msg.ID, "namespace", withs["namespace"])
					}
					continue
				}
				logged := []interface{}{"id", msg.ID, "text", string(text)}
				for k, v := range withs {
					logged = append(logged, k)
					logged = append(logged, v)
				}
				logger.Info("job update", logged...)
			case <-ctx.Done():
				break
			}
		}
	}()

	kubernetes.JobStarter(ctx, jobTriggerC, doneC)

	// Add any configured repositories to the list that need to be watched
	for i, url := range repos {
		branch := "master"
		if i < len(branches) && len(branches[i]) > 0 {
			branch = branches[i]
		}
		err = watcher.Add(url, branch, *githubToken, trackingC)
	}

	stopC := make(chan os.Signal, 1)
	signal.Notify(stopC, os.Interrupt, syscall.SIGTERM)

	// Wait until an external party indicates the server should be stopped
	<-stopC

	// Cancel the servers context in order to stop processing
	cancel()

	// This will block for a short time while waiting for an orderly shutdown
	shutdownTimeout := time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if !watcher.Stop(shutdownCtx) {
		logger.Warn("git watch had to force shutdown", "timeout", shutdownTimeout)
		os.Exit(-1)
	}
}
