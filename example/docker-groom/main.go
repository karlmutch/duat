package main

// This file contains the implementation of utility to groom older pre-release versions of
// the existing development branch and version from the local docker repository. At least the latest version will be kept.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/mgutz/logxi" // Using a forked copy of this package results in build issues

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack

	"github.com/karlmutch/duat"
	"github.com/karlmutch/duat/version"

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envfla
)

var (
	logger = logxi.New("docker-groom")

	verbose = flag.Bool("v", false, "When enabled will print internal logging for this tool")
	module  = flag.String("component", ".", "The name of the component that is being used to identify the container image, this will default to the name of the current working directory")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       docker grooming tool (docker-groom)      ", version.GitHash, "    ", version.BuildTime)
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
	if *module == "." {
		if cwd, errGo := os.Getwd(); errGo == nil {
			*module = filepath.Base(cwd)
		}
	}
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
	if !*verbose {
		logger.SetLevel(logxi.LevelError)
	}

	logger.Debug(fmt.Sprintf("%s built at %s, against commit id %s\n", os.Args[0], version.BuildTime, version.GitHash))

	cwd, errGo := os.Getwd()
	if errGo != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(errGo, "current directory unknown").With("stack", stack.Trace().TrimRuntime()).Error())
		os.Exit(-1)
	}

	md := duat.MetaData{}
	if err := md.LoadGit(cwd, true); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-2)
	}

	// The main README.md will be at the git repos top directory
	readme := filepath.Join(md.Git.Dir, "README.md")
	_, err := md.LoadVer(readme)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-2)
	}

	// Check to see if we are in a development pre-release context
	if len(md.SemVer.Prerelease()) == 0 {
		fmt.Fprintln(os.Stderr, "only pre-release versions of software containers can be groomed")
		os.Exit(-3)
	}

	// Ensure that the module name is made docker compatible
	*module, err = md.ScrubDockerRepo(*module)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-6)
	}

	// Get the git repo name and the parent which will have been used to name the
	// containers being created during our build process
	gitParts := strings.Split(md.Git.URL, "/")
	label := strings.TrimSuffix(gitParts[len(gitParts)-1], ".git")
	image := fmt.Sprintf("%s/%s/%s:%s", gitParts[len(gitParts)-2], label, *module, md.SemVer.String())

	// Now we have a pre-release make sure that it has at least 2 parts seperated by dashes.
	preParts := strings.Split(md.SemVer.Prerelease(), "-")
	if len(preParts) < 2 {
		fmt.Fprintln(os.Stderr, "only pre-release versions of software containers, with duat style date time tags, can be groomed")
		os.Exit(-6)
	}

	// Take off the duat time stamp suffix
	dockerPrefix := strings.TrimSuffix(image, preParts[len(preParts)-1])

	fmt.Fprintln(os.Stdout, dockerPrefix)

	// Now get our local docker images
	dock, errGo := client.NewEnvClient()
	if errGo != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(errGo, "docker unavailable").With("stack", stack.Trace().TrimRuntime()).Error())
		os.Exit(-4)
	}

	images, errGo := dock.ImageList(context.Background(), types.ImageListOptions{})
	if errGo != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).Error())
		os.Exit(-5)
	}

	for _, image := range images {
		for _, repo := range image.RepoTags {
			repoParts := strings.SplitN(repo, ":", 2)
			fmt.Fprintf(os.Stdout, "%+v\n", repoParts)
		}
	}
}
