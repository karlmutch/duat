// +build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/karlmutch/duat"
	"github.com/karlmutch/duat/version"
	colorable "github.com/mattn/go-colorable"

	"github.com/go-stack/stack"           // Forked copy of https://github.com/go-stack/stack
	"github.com/jjeffery/kv"              // Forked copy of https://github.com/jjeffery/kv
	logxi "github.com/karlmutch/logxi/v1" // Using a forked copy of this package results in build issues

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envflag
)

var (
	logger = logxi.NewLogger(logxi.NewConcurrentWriter(colorable.NewColorableStderr()), "build.go")

	verbose   = flag.Bool("v", false, "When enabled will print internal logging for this tool")
	recursive = flag.Bool("r", false, "When enabled this tool will visit any sub directories that contain main functions and build in each")
	userDirs  = flag.String("dirs", ".", "A comma seperated list of root directories that will be used a starting points looking for Go code, this will default to the current working directory")
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

	// First assume that the directory supplied is a code directory
	rootDirs := strings.Split(*userDirs, ",")
	dirs := []string{}

	err := kv.NewError("")

	// If this is a recursive build scan all inner directories looking for go code
	// and build it if there is code found
	//
	if *recursive {
		for _, dir := range rootDirs {
			// Will auto skip any vendor directories found
			found, err := duat.FindGoDirs(dir, []string{"main"})
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(-1)
			}
			dirs = append(dirs, found...)
		}
	} else {
		dirs = rootDirs
	}

	logger.Debug(fmt.Sprintf("%v", dirs))

	// Take the discovered directories and build them
	//
	outputs := []string{}
	localOut := []string{}

	for _, dir := range dirs {
		if localOut, err = runBuild(dir, "README.md"); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-2)
		}
		outputs = append(outputs, localOut...)
	}

	for _, output := range outputs {
		fmt.Fprintln(os.Stdout, output)
	}
}

// runBuild is used to restore the current working directory after the build itself
// has switch directories
//
func runBuild(dir string, verFn string) (outputs []string, err kv.Error) {

	logger.Info(fmt.Sprintf("building in %s", dir))

	cwd, errGo := os.Getwd()
	if errGo != nil {
		return outputs, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	outputs, err = build(dir, verFn)

	if errGo = os.Chdir(cwd); errGo != nil {
		logger.Warn("The original directory could not be restored after the build completed")
		if err == nil {
			err = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
	}

	return outputs, err
}

// build performs the default build for the component within the directory specified
//
func build(dir string, verFn string) (outputs []string, err kv.Error) {

	outputs = []string{}

	// Gather information about the current environment. also changes directory to the working area
	md, err := duat.NewMetaData(dir, verFn)
	if err != nil {
		return outputs, err
	}

	return md.GoBuild([]string{}, []string{}, "", "", true)
}
