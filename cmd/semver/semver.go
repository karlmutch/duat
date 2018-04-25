package main

// This file contains the main function for a semver version increment tool
// that is inteded for use where the CI/CD pipeline is storing the version number
// within a markdown file such as a CHANGELOG.md or README.md file
//
import (
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/karlmutch/duat"
	"github.com/karlmutch/duat/version"
	colorable "github.com/mattn/go-colorable"

	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks

	"github.com/karlmutch/semver" // Forked copy of https://github.com/Masterminds/semver

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envflag

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
	"github.com/mgutz/logxi"      // Using a forked copy of this package results in build issues
)

var (
	logger = logxi.NewLogger(logxi.NewConcurrentWriter(colorable.NewColorableStderr()), "semver")

	verFn   = flag.String("f", "README.md", "The file to be used as the source of truth for the existing, and future, version")
	applyFn = flag.String("t", "", "The files to which the version data will be propagated")
	verbose = flag.Bool("v", false, "When enabled will print internal logging for this tool")

	gitRepo = flag.String("git", ".", "The top level of the git repo to be used for the dev version")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options] [arguments]      Semantic Version tool (semver)      ", version.GitHash, "    ", version.BuildTime)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "    major                Increments the major version inside the input file")
	fmt.Fprintln(os.Stderr, "    minor                Increments the minor version inside the input file")
	fmt.Fprintln(os.Stderr, "    patch                Increments the patch version inside the input file")
	fmt.Fprintln(os.Stderr, "    pre, prerelease      Updates the pre-release version inside the input file")
	fmt.Fprintln(os.Stderr, "    apply                Propogate the version from the input file to the target files")
	fmt.Fprintln(os.Stderr, "    extract              Retrives the version tag string from the file")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "When using pre the branch name will be injected into the pre-release data along with the commit sequence number for that branch and then the commit-id.")
	fmt.Fprintln(os.Stderr, "It is possible that when using 'pre' the precedence between different developers might not be in commit strict order, but in the order that the files were processed.")
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

var (
	rFind *regexp.Regexp
	rHTML *regexp.Regexp
)

func init() {
	flag.Usage = usage

	r, errGo := regexp.Compile("\\<repo-version\\>.*?\\</repo-version\\>")
	if errGo != nil {
		fmt.Fprintf(os.Stderr, "%v\n",
			errors.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash))
		return
	}
	rFind = r
	r, errGo = regexp.Compile("<[^>]*>")
	if errGo != nil {
		fmt.Fprintf(os.Stderr, "%v\n",
			errors.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash))
		return
	}
	rHTML = r
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
		fmt.Fprintf(os.Stderr, "too many (%d - %v), command(s). you must specify only one of the commands [major|minor|patch|pre|extract]\n", len(flag.Args()), flag.Args())
		os.Exit(-1)
	}

	if _, err := os.Stat(*verFn); err != nil {
		fmt.Fprintln(os.Stderr, "the input file was not found")
		os.Exit(-2)
	}

	md := &duat.MetaData{}
	_, err := md.LoadVer(*verFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the input file version string that is currently in the file is not valid due to '%v'\n", err)
		os.Exit(-2)
	}
	ver := md.SemVer.String()

	gitErr := md.LoadGit(*gitRepo, true)

	switch flag.Arg(0) {
	case "major":
		*md.SemVer = md.SemVer.IncMajor()
	case "minor":
		*md.SemVer = md.SemVer.IncMinor()
	case "patch":
		*md.SemVer = md.SemVer.IncPatch()
	case "pre", "prerelease":
		if gitErr != nil {
			fmt.Fprintf(os.Stderr, "an operation that required git failed due to %v\n", gitErr)
			os.Exit(-5)
		}
		md.SemVer, err = md.Prerelease()
	case "apply":
		err = md.Apply(strings.Split(*applyFn, ","))
	case "", "extract":
		break
	default:
		fmt.Fprintf(os.Stderr, "invalid command, you must specify one of the commands [major|minor|patch|pre|extract], '%s' is not a valid command\n", os.Args[1])
		os.Exit(-2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "the attempt to increment the version failed due to %v\n", err)
		os.Exit(-4)
	}

	if _, errGo := semver.NewVersion(md.SemVer.String()); errGo != nil {
		fmt.Fprintf(os.Stderr, "the updated file version string generated by this tooling is not valid due to '%v'\n", errGo)
		os.Exit(-2)
	}

	// Having generated or extracted a version string if it is different as a result of processing we need
	// to update the original file
	if ver != md.SemVer.String() {
		if err := md.Replace(*verFn, *verFn, false); err != nil {
			fmt.Fprintf(os.Stderr, "the attempt to write the incremented version back failed due to %v\n", err)
			os.Exit(-4)
		}
	}

	fmt.Fprintf(os.Stdout, "%s\n", md.SemVer.String())
}
