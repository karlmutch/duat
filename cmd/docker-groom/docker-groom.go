package main

// This file contains the implementation of utility to groom older pre-release versions of
// the existing development branch and version from the local docker repository. At least the latest version will be kept.

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/mgutz/logxi" // Using a forked copy of this package results in build issues

	"github.com/karlmutch/duat"
	"github.com/karlmutch/duat/version"

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envfla
)

var (
	logger = logxi.New("docker-groom")

	verFn    = flag.String("f", "README.md", "The file to be used as the source of truth for the existing, and future, version")
	groomAll = flag.Bool("all", false, "Do not leave any images, default is false to leave the latest image for the component")
	verbose  = flag.Bool("v", false, "When enabled will print internal logging for this tool")
	module   = flag.String("module", ".", "The directory of the module that is being used to identify the container image, this will default to current working directory")
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

	md, err := duat.NewMetaData(*module, *verFn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-1)
	}

	// Check to see if we are in a development pre-release context
	if len(md.SemVer.Prerelease()) == 0 {
		fmt.Fprintln(os.Stderr, "only pre-release versions of software containers can be groomed")
		os.Exit(-2)
	}

	if err = md.ImagePrune(*groomAll); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-3)
	}
}
