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
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/karlmutch/bump-ver/version"

	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks

	"github.com/karlmutch/semver" // Forked copy of https://github.com/Masterminds/semver

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envflag

	"github.com/karlmutch/base62" // Fork of https://github.com/mattheath/base62
	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
	"github.com/mgutz/logxi"      // Using a forked copy of this package results in build issues

	"gopkg.in/src-d/go-git.v4" // Not forked due to depency tree being too complex, src-d however are a serious org so I dont expect the repo to disappear
)

var (
	logger = logxi.New("bump-ver")

	verFn   = flag.String("f", "README.md", "The file to be processed")
	applyFn = flag.String("t", "", "The files to which the version data will be propogated")
	gitRepo = flag.String("git", ".", "The top level of the git repo to be used for the dev version")
	verbose = flag.Bool("v", false, "When enabled will print internal logging for this tool")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options] [arguments]      Bump HTML Version Tag tool (bump-ver)      ", version.GitHash, "    ", version.BuildTime)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "    major    Increments the major version inside the input file")
	fmt.Fprintln(os.Stderr, "    minor    Increments the minor version inside the input file")
	fmt.Fprintln(os.Stderr, "    patch    Increments the patch version inside the input file")
	fmt.Fprintln(os.Stderr, "    dev      Updates the pre-release version inside the input file")
	fmt.Fprintln(os.Stderr, "    apply    Propogate the version from the input file to the target files")
	fmt.Fprintln(os.Stderr, "    extract  Retrives the version tag string from the file")
	fmt.Fprintln(os.Stderr, "    inject   Retrives the version tag string, then injects it into the target (-t file producing output on stdout)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "When using dev the branch name will be injected into the pre-release data along with the commit sequence number for that branch and then the commit-id.")
	fmt.Fprintln(os.Stderr, "It is possible that when using 'dev' the precedence between different developers might not be in commit strict order, but in the order that the files were processed.")
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
	rFind       *regexp.Regexp
	rHTML       *regexp.Regexp
	rVerReplace *regexp.Regexp
)

func init() {
	flag.Usage = usage

	r, errGo := regexp.Compile("\\<repo-version\\>.*?\\</repo-version\\>")
	if errGo != nil {
		fmt.Fprintf(os.Stderr, "%v",
			errors.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash))
		return
	}
	rFind = r
	r, errGo = regexp.Compile("<[^>]*>")
	if errGo != nil {
		fmt.Fprintf(os.Stderr, "%v",
			errors.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash))
		return
	}
	rHTML = r
	r, errGo = regexp.Compile("\\<repo-version\\>(.*?)\\</repo-version\\>")
	if errGo != nil {
		fmt.Fprintf(os.Stderr, "%v",
			errors.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash))
		return
	}
	rVerReplace = r
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

	if len(flag.Args()) > 1 || len(flag.Arg(0)) == 0 {
		usage()
		fmt.Fprintf(os.Stderr, "missing, or too many (%d - %v), command(s). you must specify only one of the commands [major|minor|patch|dev|extract]\n", len(flag.Args()), flag.Args())
		os.Exit(-1)
	}

	if _, err := os.Stat(*verFn); err != nil {
		fmt.Fprintf(os.Stderr, "the input file was not found")
		os.Exit(-2)
	}

	ver, err := extract(*verFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the input file could not be validated due to %v", err)
		os.Exit(-3)
	}

	semVer, errGo := semver.NewVersion(ver)
	if errGo != nil {
		fmt.Fprintf(os.Stderr, "the input file version string that is currently in the file is not valid due to '%v'", errGo)
		os.Exit(-2)
	}

	switch flag.Arg(0) {
	case "major":
		*semVer = semVer.IncMajor()
	case "minor":
		*semVer = semVer.IncMinor()
	case "patch":
		*semVer = semVer.IncPatch()
	case "dev":
		*semVer, err = dev(*semVer)
	case "apply":
		*semVer, err = apply(*semVer, strings.Split(*applyFn, ","))
	case "extract":
		break
	case "inject":
		*semVer, err = inject(*semVer, strings.Split(*applyFn, ","))
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
		if err = replace(*semVer, *verFn, *verFn, false); err != nil {
			fmt.Fprintf(os.Stderr, "the attempt to write the bumped version back failed due to %v", err)
			os.Exit(-4)
		}
	}

	if flag.Arg(0) != "inject" {
		fmt.Fprintf(os.Stdout, "%s\n", semVer.String())
	}
}

func getGitBranch(gitDir string) (branch string, err errors.Error) {
	if _, errGo := os.Stat(filepath.Join(gitDir, ".git")); err != nil {
		return "", errors.Wrap(errGo, "does not appear to be the top directory of a git repo").With("stack", stack.Trace().TrimRuntime()).With("git", gitDir)
	}

	repo, errGo := git.PlainOpen(gitDir)
	if errGo != nil {
		return "", errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("git", gitDir)
	}
	ref, errGo := repo.Head()
	if errGo != nil {
		return "", errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("git", gitDir)
	}
	splits := strings.Split(ref.Name().String(), "/")
	return splits[len(splits)-1], nil
}

func apply(semVer semver.Version, files []string) (result semver.Version, err errors.Error) {

	result = semVer

	if len(files) == 0 {
		return result, errors.New("the apply command requires that files are specified with the -t option").With("stack", stack.Trace().TrimRuntime())
	}

	checkedFiles := make([]string, 0, len(files))
	for _, file := range files {
		if len(file) != 0 {
			if _, err := os.Stat(file); err != nil {
				fmt.Fprintf(os.Stderr, "a user specified target file was not found '%s'\n", file)
				continue
			}
			checkedFiles = append(checkedFiles, file)
		}
	}

	if len(checkedFiles) != len(files) {
		fmt.Fprintln(os.Stderr, "no usable targets were found to apply the version to")
		os.Exit(-4)
	}

	// Process the files but stop on any errors
	for _, file := range checkedFiles {
		if err = replace(semVer, file, file, false); err != nil {
			return result, err
		}
	}

	return result, nil
}

func inject(semVer semver.Version, files []string) (result semver.Version, err errors.Error) {

	result = semVer

	if len(files) != 1 || len(files[0]) == 0 {
		return result, errors.New("the inject command requires that only a single target file is specified with the -t option").With("stack", stack.Trace().TrimRuntime())
	}

	if _, err := os.Stat(files[0]); err != nil {
		return result, errors.New(fmt.Sprintf("a user specified target file was not found '%s'\n", files[0])).With("stack", stack.Trace().TrimRuntime())
	}

	// Process the files but stop on any errors
	if err = replace(semVer, files[0], "-", true); err != nil {
		return result, err
	}

	return result, nil
}

func dev(semVer semver.Version) (result semver.Version, err errors.Error) {
	result = semVer
	// Generate a pre-release suffix for semver that uses a mixture of the branch name
	// with nothing but hyphens and alpha numerics, followed by a teimstamp encoded using
	// semver compatible Base62 in a way that preserves sort ordering
	//
	build := base62.EncodeInt64(time.Now().Unix())
	branch, err := getGitBranch(*gitRepo)
	if err != nil {
		return semVer, err
	}
	result, errGo := result.SetPrerelease(fmt.Sprintf("%s-%s", branch, build))
	if errGo != nil {
		return semVer, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	return result, nil
}

func extract(fn string) (ver string, err errors.Error) {
	file, errGo := os.Open(fn)
	if errGo != nil {
		return "", errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)

	for scan.Scan() {
		versions := rFind.FindAllString(scan.Text(), -1)
		if len(versions) == 0 {
			continue
		}
		if len(versions) != 1 && len(ver) != 0 {
			return "", errors.New("input file must only have a single repo-version HTML tag present").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
		}
		ver = html.UnescapeString(rHTML.ReplaceAllString(versions[0], ""))
	}
	return ver, nil
}

func replace(semVer semver.Version, fn string, dest string, substitute bool) (err errors.Error) {

	// To prevent destructive replacements first copy the file then modify the copy
	// and in an atomic operation copy the copy back over the original file, then
	// delete the working file
	origFn, errGo := filepath.Abs(fn)
	if errGo != nil {
		return errors.Wrap(errGo, "input file could not be resolved to an absolute file path").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	tmp, errGo := ioutil.TempFile(filepath.Dir(origFn), filepath.Base(origFn))
	if errGo != nil {
		return errors.Wrap(errGo, "temporary file could not be generated").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	defer func() {
		defer os.Remove(tmp.Name())

		tmp.Close()
	}()

	file, errGo := os.OpenFile(origFn, os.O_RDWR, 0600)
	if errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	newVer := fmt.Sprintf("<repo-version>%s</repo-version>", semVer.String())
	if substitute {
		newVer = fmt.Sprintf("%s", semVer.String())
	}

	scan := bufio.NewScanner(file)
	for scan.Scan() {
		tmp.WriteString(rVerReplace.ReplaceAllString(scan.Text(), newVer) + "\n")
	}

	tmp.Sync()
	if fn == dest {
		defer file.Close()
	} else {
		file.Close()

		if dest == "-" {
			file = os.Stdout
		} else {
			file, errGo = os.OpenFile(dest, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
			if errGo != nil {
				return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
			}
			defer file.Close()
		}
	}

	if dest != "-" {
		if _, errGo = file.Seek(0, io.SeekStart); errGo != nil {
			return errors.Wrap(errGo, "failed to rewind the input file").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
		}
	}
	if _, errGo = tmp.Seek(0, io.SeekStart); errGo != nil {
		return errors.Wrap(errGo, "failed to rewind a temporary file").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	// Copy the output file on top of the original file
	written, errGo := io.Copy(file, tmp)
	if errGo != nil {
		return errors.Wrap(errGo, "failed to update the input file").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	// Because we overwrote the file we need to trim off the end of the file if it shrank in size
	file.Truncate(written)

	return nil
}
