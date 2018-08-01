package main

// This file is used to expose a CLI command for creating an image from a Dockerfile

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	colorable "github.com/mattn/go-colorable"
	"github.com/mgutz/logxi" // Using a forked copy of this package results in build issues

	"github.com/karlmutch/duat"
	"github.com/karlmutch/duat/version"

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envfla
	"github.com/karlmutch/errors"  // Forked copy of https://github.com/jjeffery/errors
)

var (
	logger = logxi.NewLogger(logxi.NewConcurrentWriter(colorable.NewColorableStderr()), "image-build")

	verFn     = flag.String("f", "README.md", "The file to be used as the source of truth for the existing, and future, version")
	verbose   = flag.Bool("v", false, "When enabled will print internal logging for this tool")
	userDirs  = flag.String("dirs", ".", "A comma seperated list of root directories that will be used a starting points looking for Go code, this will default to the current working directory")
	recursive = flag.Bool("r", false, "When enabled this tool will visit any sub directories that contain a Dockerfile and build an image in each")
	module    = flag.String("module", ".", "The directory of the module that is being used to identify the container image, this will default to current working directory")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       image builder (image-build)      ", version.GitHash, "    ", version.BuildTime)
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

	repo, ver, _, err := md.GenerateImageName()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-2)
	}

	if logger.IsDebug() {
		logger.Debug(fmt.Sprintf("%s:%s", repo, ver))
	}

	// First assume that the directory supplied is a code directory
	rootDirs := strings.Split(*userDirs, ",")
	dirs := []string{}

	// If this is a recursive build scan all inner directories looking for
	// a Dockerfile and build it if one is found, otherwise this next block
	// of code just looks at each rootDir in turn

	for _, dir := range rootDirs {
		errGo := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if !*recursive {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Base(path) == "Dockerfile" {
				absDir, errGo := filepath.Abs(filepath.Dir(path))
				if errGo != nil {
					fmt.Fprintln(os.Stderr, errors.Wrap(errGo).With("dir", dir).Error())
					os.Exit(-1)
				}
				dirs = append(dirs, absDir)
			}
			return nil
		})
		if errGo != nil {
			fmt.Fprintln(os.Stderr, errors.Wrap(errGo).With("dir", dir).Error())
			os.Exit(-1)
		}

	}

	logger.Debug(fmt.Sprintf("%v", dirs))

	for _, dir := range dirs {
		logger.Info(fmt.Sprintln("building in", dir, "version", ver, "writing to", filepath.Join(dir, filepath.Base(dir)+".tar")))
		/**
				if err := md.ImageBuild(dir, "", ver, []string{}, filepath.Join(dir, filepath.Base(dir)+".tar")); err != nil {
					fmt.Fprintln(os.Stderr, err.Error())
					os.Exit(-3)
				}
		**/
	}
}
