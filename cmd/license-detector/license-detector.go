package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/karlmutch/duat/version"

	logxi "github.com/karlmutch/logxi/v1"

	"github.com/go-stack/stack"
	"github.com/jjeffery/kv" // MIT License

	fileio "github.com/karlmutch/duat/pkg/fileio"
	"github.com/karlmutch/envflag"

	"github.com/go-enry/go-license-detector/v4/licensedb"
	"github.com/go-enry/go-license-detector/v4/licensedb/filer"
)

var (
	logger = logxi.New("license-detector.go")

	shortFormOpt   = false
	inputOpt       = ""
	outputOpt      = ""
	recursivelyOpt = true
	verboseOpt     = false
)

const (
	defaultOutputFN = "license-detector.rpt"
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       license scanning and detection tool (license-detector.go)      ", version.GitHash, "    ", version.BuildTime)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments")
	fmt.Fprintln(os.Stderr, "")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Environment Variables:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "options can also be extracted from environment variables by changing dashes '-' to underscores and using upper case.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "log levels are handled by the LOGXI env variables, these are documented at https://github.com/karlmutch/logxi")
}

func init() {
	flag.BoolVar(&verboseOpt, "v", false, "Print internal logging for this tool")
	flag.BoolVar(&verboseOpt, "verbose", false, "Print internal logging for this tool")
	flag.StringVar(&inputOpt, "input", ".", "The location of files to be scanned for licenses, or a single file")
	flag.StringVar(&inputOpt, "i", ".", "The location of files to be scanned for licenses, or a single file")
	flag.BoolVar(&recursivelyOpt, "recurse", true, "Descend all directories looking for licensed source code")
	flag.BoolVar(&recursivelyOpt, "r", true, "Descend all directories looking for licensed source code")
	flag.StringVar(&outputOpt, "output", defaultOutputFN, "The location of the license detection SPDX inventory output file, the referenced directory will be skipped from license checks")
	flag.StringVar(&outputOpt, "o", defaultOutputFN, "The location of the license detection SPDX inventory output file, the referenced directory will be skipped from license checks")
	flag.BoolVar(&shortFormOpt, "short", false, "Output just a summary of the license types detected")
	flag.BoolVar(&shortFormOpt, "s", false, "Output just a summary of the license types detected")

	flag.Usage = usage
}

func main() {
	// Parse the CLI flags
	if !flag.Parsed() {
		envflag.Parse()
	}

	if verboseOpt {
		logger.SetLevel(logxi.LevelDebug)
	}

	// Check the output specification to see if it is a directory or a full file specification
	// and load defaults
	output, errGo := filepath.Abs(filepath.Clean(outputOpt))
	if errGo != nil {
		logger.Fatal(kv.Wrap(errGo, "output manifest file invalid").With("output", outputOpt).With("stack", stack.Trace().TrimRuntime()).Error())
	}

	// Assume path is to a file by default
	outputDir := filepath.Dir(output)
	outputFn := filepath.Base(output)

	// Handle the case when it is an existing directory
	if fileio.IsDir(output) {
		outputDir = output
		outputFn = defaultOutputFN
	} else {
		if !fileio.IsDir(outputDir) {
			logger.Fatal(kv.Wrap(errGo, "output manifest directory invalid").With("dir", outputDir).With("stack", stack.Trace().TrimRuntime()).Error())
		}
	}

	if err := licensesManifest(inputOpt, recursivelyOpt, outputDir, outputFn, shortFormOpt); err != nil {
		logger.Fatal(err.Error())
	}
}

type License struct {
	lic        string
	confidence float32
}

func licensesManifest(input string, recursive bool, outputDir string, outputFN string, shortForm bool) (err kv.Error) {

	output, errGo := filepath.Abs(filepath.Join(outputDir, outputFN))
	if errGo != nil {
		return kv.Wrap(errGo, "output manifest file invalid").With("stack", stack.Trace().TrimRuntime())
	}

	logger.Debug("output destination", "output", output, "stack", stack.Trace().TrimRuntime())

	// outputDir is supplied as a skipped dir as there is an issue with the library being used desending into
	// output files that have SPDX output data in them
	skipDirs := map[string]interface{}{outputDir: nil}
	allLics, err := licenses(input, recursive, skipDirs)
	if err != nil {
		return kv.Wrap(err, "could not create a license manifest").With("stack", stack.Trace().TrimRuntime())
	}

	licf, errGo := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if errGo != nil {
		return kv.Wrap(errGo, "could not create a license manifest").With("stack", stack.Trace().TrimRuntime())
	}
	defer licf.Close()

	if shortForm {
		// Find the highest confidence of each license type and output those
		summary := map[string]float32{}
		for _, lics := range allLics {
			for _, lic := range lics {
				conf, isPresent := summary[lic.lic]
				if !isPresent {
					summary[lic.lic] = lic.confidence
					continue
				}
				if conf < lic.confidence {
					summary[lic.lic] = lic.confidence
				}
			}
		}

		if logger.IsTrace() {
			logger.Trace("preparing output", "summary", spew.Sdump(summary), "stack", stack.Trace().TrimRuntime())
		}

		if len(summary) != 0 {
			lines := make([]string, 0, len(summary))
			for lic, conf := range summary {
				lines = append(lines, fmt.Sprint(lic, ",", conf))
			}
			sort.Strings(lines)
			licf.WriteString(strings.Join(lines, "\n"))
			licf.WriteString("\n")
		}

	} else {
		// Long form output
		if len(allLics) != 0 {
			// output all of the license details for files/directories, and also all license types suspected of existing
			lines := make([]string, 0, len(allLics))
			for dir, lics := range allLics {
				for _, lic := range lics {
					lines = append(lines, fmt.Sprint(dir, ",", lic.lic, ",", lic.confidence))
				}
			}

			sort.Strings(lines)
			licf.WriteString(strings.Join(lines, "\n"))
			licf.WriteString("\n")
		}
	}

	return nil
}

// licenses returns a list of directories and files that have license and confidences related to
// each.  An attempt is made to rollup results so that directories with licenses that match all
// files are aggregated into a single entry for the items, any small variations for files are
// called out and left in the output.  Also directories are rolled up where their children match.
//
func licenses(startDir string, recursively bool, skipDirs map[string]interface{}) (lics map[string][]License, err kv.Error) {

	defer func() {
		if err != nil {
			logger.Debug("failed extracting licenses", "error", err.Error(), "stack", stack.Trace().TrimRuntime())
		} else {
			logger.Debug("extracted licenses", "licenses", len(lics), "stack", stack.Trace().TrimRuntime())
		}
	}()

	logger.Debug("loading the license DB", "stack", stack.Trace().TrimRuntime())
	licensedb.Preload()

	rootDir, errGo := filepath.Abs(startDir)
	if errGo != nil {
		return lics, kv.Wrap(errGo).With("startDir", startDir).With("stack", stack.Trace().TrimRuntime())
	}
	logger.Debug("scanning", "directory", rootDir, "stack", stack.Trace().TrimRuntime())

	lics = map[string][]License{}
	dirs := []string{}
	files := []string{}

	if recursively {
		// Get a list of all of the directory names recursively
		filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				dirs = append(dirs, path)
				logger.Trace("including", "directory", path, "stack", stack.Trace().TrimRuntime())
			}
			return nil
		})
	} else {
		if fileio.IsDir(rootDir) {
			dirs = append(dirs, rootDir)
			logger.Trace("including", "directory", rootDir, "stack", stack.Trace().TrimRuntime())
		} else {
			files = append(files, rootDir)
		}
	}

	logger.Debug("processing", "directories", len(dirs), "files", len(files), "stack", stack.Trace().TrimRuntime())

	for _, dir := range dirs {
		fr, errGo := filer.FromDirectory(dir)
		if errGo != nil {
			return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
		logger.Trace("processing", "dir", dir, "stack", stack.Trace().TrimRuntime())
		licenses, errGo := licensedb.Detect(fr)
		if errGo != nil && errGo.Error() != "no license file was found" {
			return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
		logger.Trace("processed", "dir", dir, "licenses", len(licenses), "stack", stack.Trace().TrimRuntime())

		// Include directories with no licenses so the auditing software knows we still scanned them
		if _, isPresent := lics[dir]; !isPresent {
			lics[dir] = []License{}
		}
		if len(licenses) != 0 {
			for lic, match := range licenses {
				lics[dir] = append(lics[dir], License{lic: lic, confidence: match.Confidence})
			}
			sort.Slice(lics[dir], func(i, j int) bool { return lics[dir][i].confidence < lics[dir][j].confidence })
		}
		logger.Trace("recorded", "dir", dir, "licenses", len(licenses), "stack", stack.Trace().TrimRuntime())
	}

	if len(files) == 0 {
		fr, errGo := FromFiles(files)
		if errGo != nil {
			return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
		logger.Trace("processing", "files", len(files), "stack", stack.Trace().TrimRuntime())
		licenses, errGo := licensedb.Detect(fr)
		if errGo != nil && errGo.Error() != "no license file was found" {
			return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
		logger.Trace("processed", "files", len(files), "licenses", len(licenses), "stack", stack.Trace().TrimRuntime())

		// Include directories with no licenses so the auditing software knows we still scanned them
		for _, aFile := range files {
			lics[aFile] = []License{}
		}
		if len(licenses) != 0 {
			for lic, match := range licenses {
				lics[lic] = append(lics[lic], License{lic: lic, confidence: match.Confidence})
			}
		}
		for aFile, _ := range lics {
			sort.Slice(lics[aFile], func(i, j int) bool { return lics[aFile][i].confidence < lics[aFile][j].confidence })
		}
		logger.Trace("recorded", "files", len(files), "licenses", len(licenses), "stack", stack.Trace().TrimRuntime())
	}

	return lics, nil
}
