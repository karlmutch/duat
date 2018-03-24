package main

// This file contains the implementation of a CLI to access github and prepare releases
// of binaries etc for a release using the current semantic version as the released version

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mgutz/logxi" // Using a forked copy of this package results in build issues

	"github.com/karlmutch/duat"
	"github.com/karlmutch/duat/version"

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envfla

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

var (
	logger = logxi.New("github-release")

	verFn        = flag.String("f", "README.md", "The file to be used as the source of truth for the existing, and future, version")
	verbose      = flag.Bool("v", false, "When enabled will print internal logging for this tool")
	externalRepo = flag.String("release-repo", "", "The name of a remote image repository, this will default to no remote repo")
	token        = flag.String("github-token", "", "The github token string obtained from https://github.com/settings/tokens, defaults to the env var GUTHUB_TOKEN")
	message      = flag.String("description", "", "A text description of the release")
	module       = flag.String("module", ".", "The name of the component that is being used to identify the container image, this will default to the current working directory")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       github release tool (github-release)      ", version.GitHash, "    ", version.BuildTime)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "    Supply a list of filenames, or wildcards, as arguments to include them into the release.")
	fmt.Fprintln(os.Stderr, "    If you wish to have the list of files names supplied using shell pipes then use a dash '-' as the argument.")
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

func init() {
	flag.Usage = usage
}

func main() {

	// Parse the CLI flags
	if !flag.Parsed() {
		envflag.Parse()
	}

	// Turn off logging regardless of the default levels if the verbose flag is not enabled.
	// By design this is a CLI tool and outputs information that is expected to be used by shell
	// scripts etc
	//
	if *verbose {
		logger.SetLevel(logxi.LevelDebug)
	}
	if len(os.Args) < 1 {
		fmt.Fprintln(os.Stderr, errors.New("no input file names were supplied").With("stack", stack.Trace().TrimRuntime()))
		os.Exit(-1)
	}

	fns := []string{}
	for _, fn := range os.Args[1:] {
		if fn == "-" {
			buff, errGo := ioutil.ReadAll(os.Stdin)
			if errGo != nil {
				fmt.Fprintln(os.Stderr, errors.Wrap(errGo, "no input file names were supplied").With("stack", stack.Trace().TrimRuntime()))
				os.Exit(-1)
			}
			fns = append(fns, strings.Split(string(buff), "\n")...)
		} else {
			fns = append(fns, fn)
		}
	}

	uploads := []string{}
	for _, fn := range fns {
		files, errGo := filepath.Glob(fn)
		if errGo != nil {
			fmt.Fprintln(os.Stderr, errors.Wrap(errGo, "no input file names were supplied").With("stack", stack.Trace().TrimRuntime()))
			os.Exit(-1)
		}
		uploads = append(uploads, files...)
	}

	if len(uploads) == 0 {
		fmt.Fprintln(os.Stderr, errors.New("input file wildcards did not match any files").With("stack", stack.Trace().TrimRuntime()))
		os.Exit(-1)
	}

	logger.Debug(fmt.Sprintf("%s built at %s, against commit id %s\n", os.Args[0], version.BuildTime, version.GitHash))

	md, err := duat.NewMetaData(*module, *verFn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-1)
	}

	if err = md.CreateRelease(*token, "", uploads); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-1)
	}
}
