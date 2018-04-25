package main

// This file is used to expose a CLI command for testing if an image exists or not for the current build tree

import (
	"flag"
	"fmt"
	"os"
	"path"

	colorable "github.com/mattn/go-colorable"
	"github.com/mgutz/logxi" // Using a forked copy of this package results in build issues

	"github.com/karlmutch/duat"
	"github.com/karlmutch/duat/version"

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envfla
)

var (
	logger = logxi.NewLogger(logxi.NewConcurrentWriter(colorable.NewColorableStderr()), "image-exists")

	verFn   = flag.String("f", "README.md", "The file to be used as the source of truth for the existing, and future, version")
	verbose = flag.Bool("v", false, "When enabled will print internal logging for this tool")
	module  = flag.String("module", ".", "The name of the component that is being used to identify the container image, this will default to the current working directory")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       image exists test tool (image-exists)      ", version.GitHash, "    ", version.BuildTime)
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

	logger.Debug(fmt.Sprintf("%s built at %s, against commit id %s\n", os.Args[0], version.BuildTime, version.GitHash))

	md, err := duat.NewMetaData(*module, *verFn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-1)
	}

	if logger.IsDebug() {
		repo, ver, _, err := md.GenerateImageName()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-2)
		}
		logger.Debug(fmt.Sprintf("%s:%s", repo, ver))
	}

	exists, _, err := md.ImageExists()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-3)
	}

	if !exists {
		os.Exit(1)
	}
}
