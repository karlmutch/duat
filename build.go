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
	"github.com/mgutz/logxi"      // Using a forked copy of this package results in build issues

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envflag
)

var (
	logger = logxi.New("build.go")

	verbose   = flag.Bool("v", false, "When enabled will print internal logging for this tool")
	recursive = flag.Bool("r", false, "When enabled this tool will visit any sub directories that contain main functions and build in each")
	module    = flag.String("module", ".", "The location of the component that is being used to identify the container image, this will default to the current working directory")
	imageOnly = flag.Bool("image-only", false, "Used to only perform a docker build of the component with no other steps")

	goPath = os.Getenv("GOPATH")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       build tool (build.go)      ", version.GitHash, "    ", version.BuildTime)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments")
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

func findDirs(dir string) (dirs []string, err errors.Error) {
	dirs = []string{}

	errGo := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	if errGo != nil {
		return nil, errors.Wrap(errGo).With("dir", dir).With("stack", stack.Trace().TrimRuntime())
	}
	return dirs, err
}

func main() {
	// This code is run in the same fashion as a script and should be co-located
	// with the component that is being built

	// Parse the CLI flags
	if !flag.Parsed() {
		envflag.Parse()
	}

	if *verbose {
		logger.SetLevel(logxi.LevelDebug)
	}

	dirs := []string{*module}

	if *recursive {
		found, err := findDirs(*module)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(-2)
		}

		if found, err = duat.FindPossibleFunc("main", found); err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(-2)
		}
		dirs = nil
		for _, dir := range found {
			dirs = append(dirs, filepath.Dir(dir))
		}
	}

	logger.Debug(fmt.Sprintf("%v", dirs))

	for _, dir := range dirs {
		if err := runBuild(dir); err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(-1)
		}
	}
}

func runBuild(dir string) (err errors.Error) {

	logger.Info(fmt.Sprintf("building in %s", dir))

	cwd, errGo := os.Getwd()
	if errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	err = build(dir)

	if errGo = os.Chdir(cwd); errGo != nil {
		logger.Warn("The original directory could not be restored after the build completed")
		if err == nil {
			err = errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
	}

	return err
}

func build(dir string) (err errors.Error) {
	// Gather information about the current environment. also changes directory to the working area
	md, err := duat.NewMetaData(dir)
	if err != nil {
		return err
	}

	// If there is a Dockerfile for this module then check the images etc
	if _, err := os.Stat("./Dockerfile"); err == nil {
		if runtime, _ := md.ContainerRuntime(); len(runtime) == 0 {
			exists, _, err := md.ImageExists()
			if err != nil {
				return err
			}
			if exists {
				return errors.New("an image already exists at the current software version, using 'semver pre' to bump your pre-release version will correct this").With("stack", stack.Trace().TrimRuntime())
			}
		}
		logger.Debug("Dockerfile found and validated")
	}

	if !*imageOnly {
		// Copy the compiled file into the GOPATH bin directory
		if len(goPath) == 0 {
			return errors.New("unable to determine the compiler bin output dir, env var GOPATH might be missing or empty").With("stack", stack.Trace().TrimRuntime())
		}

		if err = md.GoCompile(); err != nil {
			return err
		}

		if errGo := os.MkdirAll(filepath.Join(goPath, "bin"), os.ModePerm); errGo != nil {
			if !os.IsExist(errGo) {
				return errors.Wrap(errGo, "unable to create the $GOPATH/bin directory").With("stack", stack.Trace().TrimRuntime())
			}
		}

		// Find any executables we have and copy them to the gopath bin directory as well
		binPath, errGo := filepath.Abs(filepath.Join(".", "bin"))
		if errGo != nil {
			return errors.Wrap(errGo, "unable to copy binary files from the ./bin directory").With("stack", stack.Trace().TrimRuntime())
		}

		filepath.Walk(binPath, func(path string, f os.FileInfo, errGo error) error {
			if f.IsDir() {
				return nil
			}
			// Is the file executable at all ?
			if f.Mode()&0111 != 0 {
				src := filepath.Join("bin", f.Name())
				dst := filepath.Join(goPath, "bin", filepath.Base(f.Name()))

				logger.Debug(fmt.Sprintf("copy bin artifact %s %s", src, dst))

				if err = duat.CopyFile(src, dst); err != nil {
					fmt.Fprintf(os.Stderr, err.Error())
					os.Exit(-6)
				}
			}
			return nil
		})
	}

	// If we have a Dockerfile in our target directory build it
	if _, err := os.Stat("Dockerfile"); err == nil {
		// Create an image
		if err := md.ImageCreate(); err != nil {
			if errors.Cause(err) == duat.ErrInContainer {
				// This only a real error if the user explicitly asked for the image to be produced
				if *imageOnly {
					return errors.New("-image-only used but we were running inside a container which is not supported").With("stack", stack.Trace().TrimRuntime())
				}
			} else {
				return err
			}
		}
		if err := md.ImagePrune(false); err != nil {
			fmt.Fprintln(os.Stderr, err.With("msg", "prune operation failed, and ignored").Error())
		}
	} else {
		if *imageOnly {
			return errors.New("-image-only used however there is no Dockerfile present").With("stack", stack.Trace().TrimRuntime())
		}
		logger.Debug(fmt.Sprintf("no Dockerfile found, image build step skipped"))
	}
	return nil
}
