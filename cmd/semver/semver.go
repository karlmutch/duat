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
	"strings"

	"github.com/jjeffery/kv"
	"github.com/karlmutch/duat"
	duatgit "github.com/karlmutch/duat/pkg/git"
	"github.com/karlmutch/duat/version"

	colorable "github.com/mattn/go-colorable"

	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks

	"github.com/Masterminds/semver"

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envflag

	logxi "github.com/karlmutch/logxi/v1" // Using a forked copy of this package results in build issues
)

var (
	logger = logxi.NewLogger(logxi.NewConcurrentWriter(colorable.NewColorableStderr()), "semver")

	verFn      = flag.String("f", "README.md,README.adoc", "A list of files from which the first match will be used as the source of truth for the existing, and any new, version")
	applyFn    = flag.String("t", "", "The files to which the version data will be propagated")
	verbose    = flag.Bool("v", false, "When enabled will print internal logging for this tool")
	prefix     = flag.String("p", "", "Decorate semver output with a user specified prefix")
	useGitTags = flag.Bool("g", false, "Use the latest Git repository tag as the input for version(s) information")
	useRCTags  = flag.Bool("rc", false, "Do not use release candidate tags when sorting, only applies to sorting")

	gitRepo = flag.String("git", ".", "The top level of the git repo to be used for the dev version")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options] [command]      Semantic Version tool (semver)      ", version.GitHash, "    ", version.BuildTime)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "    major                Increments the major version")
	fmt.Fprintln(os.Stderr, "    minor                Increments the minor version")
	fmt.Fprintln(os.Stderr, "    patch                Increments the patch version")
	fmt.Fprintln(os.Stderr, "    pre, prerelease      Updates the pre-release version")
	fmt.Fprintln(os.Stderr, "    rc, releasecandidate Updates the version to reflect the latest release candidate for the plain semver, uses the origin tags to determine the new value")
	fmt.Fprintln(os.Stderr, "    apply                Propogate the version from the version to the target files")
	fmt.Fprintln(os.Stderr, "    extract              Retrives the version tag string")
	fmt.Fprintln(os.Stderr, "    sort                 Retrives all known git tags and sorts them in semver order, ascending, and outputs them to stdout")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "When using pre the branch name will be injected into the pre-release data along with the commit sequence number for that branch and then the commit-id.")
	fmt.Fprintln(os.Stderr, "It is possible that when using 'pre' the precedence between different developers might not be in commit strict order, but in the order that the files were processed.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "If the -g option is specified then the git repository will be searched for semver tags and these will be used to determine the starting version ")
	fmt.Fprintln(os.Stderr, "prior to applying the commands")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Environment Variables:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "options can also be extracted from environment variables by changing dashes '-' to underscores and using upper case.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "log levels are handled by the LOGXI env variables, these are documented at https://github.com/mgutz/logxi")
}

func main() {

	flag.Usage = usage

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

	md := &duat.MetaData{}

	// Look for tags using the git tag history and load them if found into the project metadata
	history, err := duatgit.TagHistory()
	if *useGitTags && err != nil {
		fmt.Fprintln(os.Stderr, "no input file was found, using git tags also failed (", err.Error(), ")")
		os.Exit(-2)
	}
	if *useGitTags {
		// Use the latest tags in sorted semver order
		md.SemVer = history.Tags[len(history.Tags)-1].Tag
	}

	verFile := ""
	candidates := strings.Split(*verFn, ",")
	for _, verFile = range candidates {
		if _, err := os.Stat(verFile); err == nil {
			break
		}
	}

	if len(verFile) == 0 && !*useGitTags {
		fmt.Fprintln(os.Stderr, "no input file was found")
		os.Exit(-2)
	}

	if md.SemVer == nil {
		_, err = md.LoadVer(verFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "the input file version string that is currently in %s is not valid due to '%v'\n", verFile, err)
			os.Exit(-2)
		}
	}

	gitErr := md.LoadGit(*gitRepo, true)

	// Save the original version to determine if it needs to be applied
	ver := md.SemVer.String()

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
	case "rc", "releasecandidate":
		warnings := []kv.Error{}
		md.SemVer, err, warnings = md.IncRC()
		if err != nil {
			for _, warn := range warnings {
				fmt.Fprintln(os.Stderr, warn.Error())
			}
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-6)
		}
	case "apply":
		if len(*prefix) != 0 {
			newVer, errGo := semver.NewVersion(*prefix + md.SemVer.String())
			if errGo != nil {
				fmt.Fprintf(os.Stderr, "the attempt to write the incremented version back failed due to %v\n", err)
				os.Exit(-8)
			}
			md.SemVer = newVer
		}
		err = md.Apply(strings.Split(*applyFn, ","))
	case "", "extract":
		break
	case "sort":
		if !*useGitTags {
			fmt.Fprintln(os.Stderr, "the -g flag must be used when sorting tags")
			os.Exit(-8)
		}
		for _, aTag := range history.Tags {
			if !*useRCTags && len(aTag.Tag.Prerelease()) != 0 {
				continue
			}
			fmt.Println(aTag.Tag.String())
		}
		os.Exit(0)
		break
	default:
		fmt.Fprintf(os.Stderr, "invalid command, you must specify one of the commands [major|minor|patch|pre|extract|apply], '%s' is not a valid command\n", os.Args[1])
		os.Exit(-2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "the attempt to increment the version failed due to %v\n", err)
		os.Exit(-4)
	}

	// Check for prefix processing and if there is any then reconstruct the version with the prefix so that
	// the Original returns the non semver 2 variant, if not use proper semver 2.0
	if len(*prefix) != 0 {
		semVer, errGo := semver.NewVersion(*prefix + md.SemVer.String())
		if errGo != nil {
			fmt.Fprintf(os.Stderr, "the updated file version string generated by this tooling is not valid due to '%v'\n", errGo)
			os.Exit(-2)
		}
		md.SemVer = semVer
	}

	// Having generated or extracted a version string if it is different as a result of processing we need
	// to update the original file
	if ver != md.SemVer.String() {
		if err := md.Replace(verFile, verFile, false); err != nil {
			fmt.Fprintf(os.Stderr, "the attempt to write the incremented version back failed due to %v\n", err)
			os.Exit(-4)
		}
	}

	fmt.Fprintf(os.Stdout, "%s\n", md.SemVer.Original())
}
