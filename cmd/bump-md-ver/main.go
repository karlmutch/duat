package main

// This file contains the main function for a semver version bumping tool
// that is inteded for use where the CI/CD pipeline is storing the version number
// within a markdown file such as a CHANGELOG.md or README.md file
//
import (
	"bufio"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/karlmutch/bump-md-ver/version"
	"github.com/karlmutch/semver"

	"github.com/karlmutch/envflag"

	"github.com/go-stack/stack"
	"github.com/karlmutch/errors"

	"github.com/mgutz/logxi"
)

var (
	logger = logxi.New("bump-md-ver")

	mdFile  = flag.String("f", "README.md", "The markdown file to be processed")
	verbose = flag.Bool("v", false, "When enabled will print internal logging for this tool")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[arguments]      Bump MarkDown Version tool (bump-md-ver)      ", version.GitHash, "    ", version.BuildTime)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "    major    Increments the major version inside the MD file, and returns the new version string")
	fmt.Fprintln(os.Stderr, "    minor    Increments the minor version inside the MD file, and returns the new version string")
	fmt.Fprintln(os.Stderr, "    patch    Increments the patch version inside the MD file, and returns the new version string")
	fmt.Fprintln(os.Stderr, "    dev      Updates the pre-release meta-data version inside the MD file, and returns the new version string")
	fmt.Fprintln(os.Stderr, "    extract  Retrives the version from the markdown file")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "When using dev the branch name will be injected into the pre-release data along with the commit sequence number for that branch and then the commit-id.")
	fmt.Fprintln(os.Stderr, "It is possible that when using 'dev' the precedence between different developers might not be in commit strict order")
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

	if len(os.Args) != 2 || len(os.Args[1]) == 0 {
		usage()
		fmt.Fprintf(os.Stderr, "missing command, you must specify one of the commands [major|minor|patch|dev|extract]")
		os.Exit(-1)
	}

	if _, err := os.Stat(*mdFile); err != nil {
		fmt.Fprintf(os.Stderr, "the input file was not found")
		os.Exit(-2)
	}

	ver, err := extract(*mdFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the input file could not be validated due to %v", err)
		os.Exit(-3)
	}

	semVer, errGo := semver.NewVersion(ver)
	if errGo != nil {
		fmt.Fprintf(os.Stderr, "the input file version string that is currently in the file is not valid due to '%v'", errGo)
		os.Exit(-2)
	}

	switch os.Args[1] {
	case "major":
		err = major(semVer, *mdFile)
	case "minor":
		err = minor(semVer, *mdFile)
	case "patch":
		err = patch(semVer, *mdFile)
	case "dev":
		err = dev(semVer, *mdFile)
	case "extract":
		break
	default:
		fmt.Fprintf(os.Stderr, "invalid command, you must specify one of the commands [major|minor|patch|dev|extract], '%s' is not a valid command", os.Args[1])
		os.Exit(-2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "the attempt to bump the version failed due to %v", err)
		os.Exit(-4)
	}

	// Having generated or extracted a version string if it is different as a result of processing we need
	// to update the original file
	if ver != semVer.String() {
		if err = replace(*semVer, *mdFile); err != nil {
			fmt.Fprintf(os.Stderr, "the attempt to write the bumped version back failed due to %v", err)
			os.Exit(-4)
		}
	}

	fmt.Fprintf(os.Stdout, "%s\n", semVer.String())
}

func major(semVer *semver.Version, fn string) (err errors.Error) {
	semVer.IncMajor()
	return nil
}

func minor(semVer *semver.Version, fn string) (err errors.Error) {
	semVer.IncMinor()
	return nil
}

func patch(semVer *semver.Version, fn string) (err errors.Error) {
	semVer.IncPatch()
	return nil
}

func dev(semVer *semver.Version, fn string) (err errors.Error) {
	// TODO parse and modify the pre-release string
	return nil
}

func extract(fn string) (ver string, err errors.Error) {
	file, errGo := os.Open(fn)
	if errGo != nil {
		return "", errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With(fn)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)

	r, _ := regexp.Compile("\\<repo-version\\>.*\\</repo-version\\>")
	rHTML, _ := regexp.Compile("<[^>]*>")

	for scan.Scan() {
		versions := r.FindAllString(scan.Text(), -1)
		if len(versions) == 0 {
			continue
		}
		if len(versions) != 1 && len(ver) != 0 {
			return "", errors.New("input file must only have a single repo-version HTML tag present").With("stack", stack.Trace().TrimRuntime()).With(fn)
		}
		ver = html.UnescapeString(rHTML.ReplaceAllString(versions[0], ""))
	}
	return ver, nil
}

func replace(semVer semver.Version, fn string) (err errors.Error) {
	// To prevent destructive replacements first copy the file then modify the copy
	// and in an atomic operation copy the copy back over the original file, then
	// delete the working file
	origFn, errGo := filepath.Abs(fn)
	if errGo != nil {
		return errors.Wrap(errGo, "input file could not be resolved to an absolute file path").With("stack", stack.Trace().TrimRuntime()).With(fn)
	}
	tmp, errGo := ioutil.TempFile(filepath.Dir(origFn), filepath.Base(origFn))
	if errGo != nil {
		return errors.Wrap(errGo, "temporary file could not be generated").With("stack", stack.Trace().TrimRuntime()).With(fn)
	}
	defer tmp.Close()
	defer os.Remove(tmp.Name())

	file, errGo := os.Open(origFn)
	if errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With(fn)
	}
	defer file.Close()

	scan := bufio.NewScanner(file)
	for scan.Scan() {
	}

	// Copy the output file on top of the original file
	return nil
}
