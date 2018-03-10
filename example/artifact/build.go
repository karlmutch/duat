// +build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/karlmutch/duat"
	"github.com/karlmutch/duat/version"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envflag
)

var (
	module    = flag.String("module", ".", "The location of the component that is being used to identify the container image, this will default to the current working directory")
	imageOnly = flag.Bool("image-only", false, "Used to only perform a docker build of the component with no other steps")

	goPath = os.Getenv("GOPATH")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       build tool (build.go)      ", version.GitHash, "    ", version.BuildTime)
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
	// This code is run in the same fashion as a script and should be co-located
	// with the component that is being built

	// Parse the CLI flags
	if !flag.Parsed() {
		envflag.Parse()
	}

	// Gather information about the current environment
	md, err := duat.NewMetaData(*module)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(-1)
	}

	if runtime, _ := md.ContainerRuntime(); len(runtime) == 0 {
		exists, err := md.ImageExists()
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(-7)
		}
		if exists {
			fmt.Fprintf(os.Stderr, "an image already exists at the current software version, using 'semver pre' to bump your pre-release version will correct this\n")
			os.Exit(-8)
		}
	}

	if !*imageOnly {
		// Copy the compiled file into the GOPATH bin directory
		if len(goPath) == 0 {
			fmt.Fprintln(os.Stderr, errors.New("unable to determine the compiler bin output dir, env var GOPATH might be missing or empty").With("stack", stack.Trace().TrimRuntime()).Error())
			os.Exit(-5)
		}

		if err = md.GoCompile(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-4)
		}

		if errGo := os.MkdirAll(filepath.Join(goPath, "bin"), os.ModePerm); errGo != nil {
			if !os.IsExist(errGo) {
				fmt.Fprintln(os.Stderr, errors.Wrap(errGo, "unable to create the $GOPATH/bin directory").With("stack", stack.Trace().TrimRuntime()).Error())
				os.Exit(-2)
			}
		}

		if err = duat.CopyFile("bin/artifact", filepath.Join(goPath, "bin", "artifact")); err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(-6)
		}
	}

	// If we have a Dockerfile in our local directory build it
	if _, err := os.Stat("./Dockerfile"); err == nil {
		// Create an image
		if err := md.ImageCreate(); err != nil {
			if errors.Cause(err) == duat.ErrInContainer {
				// This only a real error if the user explicitly asked for the image to be produced
				if *imageOnly {
					fmt.Fprintln(os.Stderr, errors.New("-image-only used but we were running inside a container which is not supported").With("stack", stack.Trace().TrimRuntime()).Error())
					os.Exit(-9)
				}
			} else {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(-10)
			}
		}
		if err := md.ImagePrune(false); err != nil {
		}
	} else {
		if *imageOnly {
			fmt.Fprintln(os.Stderr, errors.New("-image-only used but there is no Dockerfile present").With("stack", stack.Trace().TrimRuntime()).Error())
			os.Exit(-11)
		}
		fmt.Fprintln(os.Stderr, "docker build step skipped")
	}
}
