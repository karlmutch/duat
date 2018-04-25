package main

// This file is used to expose a CLI command for promoting docker images

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"

	colorable "github.com/mattn/go-colorable"
	"github.com/mgutz/logxi" // Using a forked copy of this package results in build issues

	"github.com/karlmutch/duat"
	"github.com/karlmutch/duat/version"

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envfla

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
)

var (
	logger = logxi.NewLogger(logxi.NewConcurrentWriter(colorable.NewColorableStderr()), "image-release")

	verFn        = flag.String("f", "README.md", "The file to be used as the source of truth for the existing, and future, version")
	verbose      = flag.Bool("v", false, "When enabled will print internal logging for this tool")
	production   = flag.Bool("production", false, "When enabled will generate tools etc as production releases by removing pre-release version markers")
	module       = flag.String("module", ".", "The name of the component that is being used to identify the container image, this will default to the current working directory")
	externalRepo = flag.String("release-repo", "", "The name of a remote image repository, this will default to no remote repo")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       docker image release tool (image-release)      ", version.GitHash, "    ", version.BuildTime)
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

func Released(md *duat.MetaData) (err errors.Error) {
	return md.Replace(md.VerFile, md.VerFile, false)
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

	if *production {
		semVer, errGo := md.SemVer.SetPrerelease("")
		if errGo != nil {
			fmt.Fprintln(os.Stderr, "could not clear the prerelease version", errGo.Error())
			os.Exit(-2)
		}
		md.SemVer = &semVer
		if err = md.Replace(md.VerFile, md.VerFile, false); err != nil {
			fmt.Fprintln(os.Stderr, "could not save the production version", err.Error())
			os.Exit(-2)
		}
	}

	if len(*externalRepo) == 0 {
		url, err := duat.GetECRDefaultURL()
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not locate a default AWS ECR to promote the image to", err.Error())
			os.Exit(-2)
		}
		if url == nil {
			fmt.Fprintln(os.Stderr, "could not locate a default AWS ECR to promote the image to")
			os.Exit(-2)
		}
		*externalRepo = url.Hostname()
	}

	images, err := md.ImageRelease(*externalRepo, *production)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-2)
	}

	b, errGo := json.Marshal(images)
	if errGo != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-3)
	}
	fmt.Fprintln(os.Stderr, string(b))

	// If everything worked out for pushing the image etc then formally save
	// the non pre-release version
	//
	if err = Released(md); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-4)
	}
}
