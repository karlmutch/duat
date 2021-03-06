package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	logxi "github.com/karlmutch/logxi/v1" // Using a forked copy of this package results in build issues
	colorable "github.com/mattn/go-colorable"

	"github.com/karlmutch/duat"
	"github.com/karlmutch/duat/version"

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envflag
)

var (
	logger = logxi.NewLogger(logxi.NewConcurrentWriter(colorable.NewColorableStderr()), "stencil")

	verFn   = flag.String("f", "README.md,README.adoc", "A list of files from which the first match will be used as the source of truth for the existing, and any new, version")
	verbose = flag.Bool("v", false, "When enabled will print internal logging for this tool")
	module  = flag.String("module", ".", "The name of the component that is being used to identify the container image, this will default to the current working directory")

	input  = flag.String("input", "", "The name of an input file, defaults to the standard input of the shell")
	output = flag.String("output", "", "The name of an output file, default to the console")

	values       = flag.String("values", "", "A comma seperated list of k=v pairs, that can act as overriden values or new values within the template")
	suppressWarn = flag.Bool("supress-warnings", false, "This flag will cause warnings to become supressed")
	warnError    = flag.Bool("error-warnings", false, "This flag will cause warnings to be treated error exits")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       templating tool (stencil)      ", version.GitHash, "    ", version.BuildTime)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "stencil is used to process input through a templating engine, a wide range of capabilities are")
	fmt.Fprintln(os.Stderr, "available and are documented at, https://golang.org/pkg/text/template/.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Additional functions within templating are supported and documented here, ")
	fmt.Fprintln(os.Stderr, "http://masterminds.github.io/sprig/")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Variables related to versioning and git are also made accessible to template files")
	fmt.Fprintln(os.Stderr, "from the local git and semver data sources including: {{.duat.version}}, {{duat.module}}, {{.duat.[gitTag,gitHash,gitBranch,gitRepo,gitDir]}}.")
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

	verFile := ""
	candidates := strings.Split(*verFn, ",")
	for _, verFile = range candidates {

		if _, err := os.Stat(verFile); err == nil {
			break
		}
	}

	if len(verFile) == 0 {
		fmt.Fprintln(os.Stderr, "no input file was found")
		os.Exit(-2)
	}

	md, err := duat.NewMetaData(*module, verFile)
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

	// Setup the I/O streams that will be processed using the standard input
	// and output as the defaults
	in := os.Stdin
	out := os.Stdout

	if len(*input) != 0 {
		file, errGo := os.Open(*input)
		if errGo != nil {
			fmt.Fprintln(os.Stderr, errGo.Error())
			os.Exit(-2)
		}
		in = file
	}
	if len(*output) != 0 {
		file, errGo := os.Open(*output)
		if errGo != nil {
			fmt.Fprintln(os.Stderr, errGo.Error())
			os.Exit(-2)
		}
		out = file
	}

	// Take the open streams that will be used and place them into a parameter structure for the
	// template function, along with key value pairs of template variables specified on the command line
	opts := duat.TemplateOptions{
		IOFiles: []duat.TemplateIOFiles{{
			In:  in,
			Out: out,
		}},
		OverrideValues: map[string]string{},
	}

	if len(*values) != 0 {
		kvs := strings.Split(*values, ",")
		for _, kv := range kvs {
			pair := strings.SplitN(kv, "=", 2)
			if len(pair) != 2 {
				fmt.Fprintln(os.Stderr, "the key value pairs in your -values option must be seperated with an equals (=)")
				os.Exit(-3)
			}
			opts.OverrideValues[pair[0]] = pair[1]
		}
	}

	// Perform the templating operation
	err, warnings := md.Template(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-2)
	}

	// cmd line options to control the response of the stencil software to
	// warnings
	if !*suppressWarn {
		for _, err = range warnings {
			_ = logger.Warn(err.Error())
		}
	}
	if *warnError {
		os.Exit(-1)
	}
}
