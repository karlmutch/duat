package duat

// This file contains methods for Go builds using the duat conventions

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go/ast"
	"go/parser"
	"go/token"

	"github.com/go-stack/stack" // Forked copy of https://github.com/go-stack/stack
	"github.com/jjeffery/kv"    // Forked copy of https://github.com/jjeffery/kv
)

// Look for directories inside the root 'dir' and return their paths, skip any vendor directories
//
func findDirs(dir string) (dirs []string, err kv.Error) {
	dirs = []string{}

	errGo := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}
		if strings.HasPrefix(path, "vendor/") || info.Name() == "vendor" || info.Name() == ".git" {
			return filepath.SkipDir
		}
		dirs = append(dirs, path)
		return nil
	})
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("dir", dir).With("stack", stack.Trace().TrimRuntime())
	}
	return dirs, err
}

func FindGoDirs(dir string, funcs []string) (dirs []string, err kv.Error) {
	dirs = []string{}

	found, err := findDirs(dir)
	if err != nil {
		return []string{}, err
	}

	groomed, err := FindPossibleGoFuncs(funcs, found, []string{})
	if err != nil {
		return []string{}, err
	}

	for _, dir := range groomed {
		dirs = append(dirs, filepath.Dir(dir))
	}
	return dirs, nil
}

func FindGoFiles(dir string) (files []string, err kv.Error) {
	files = []string{}

	if stat, errGo := os.Stat(dir); errGo != nil {
		return files, kv.Wrap(errGo).With("dir", dir).With("stack", stack.Trace().TrimRuntime())
	} else {
		if !stat.IsDir() {
			if filepath.Ext(stat.Name()) == ".go" {
				files = append(files, dir)
			} else {
				filepath.Walk(dir, func(path string, f os.FileInfo, errGo error) error {
					if !f.IsDir() {
						if filepath.Ext(f.Name()) == ".go" {
							files = append(files, f.Name())
						}
					}
					return nil
				})
			}
		}
		return files, nil
	}
}

func GoFileTags(fn string, tags []string) (tagsSatisfied bool) {
	file, errGo := os.Open(fn)
	if errGo != nil {
		return false
	}
	defer file.Close()

	scan := bufio.NewScanner(file)
	scan.Scan()
	tokens := strings.Split(scan.Text(), " ")
	if tokens[0] != "//" {
		return true
	}
	fileTags := map[string]struct{}{}
	for _, token := range tokens[1:] {
		if len(token) != 0 {
			fileTags[token] = struct{}{}
		}
	}
	if _, isPresent := fileTags["+build"]; !isPresent {
		return true
	}
	delete(fileTags, "+build")
	for _, aTag := range tags {
		delete(fileTags, aTag)
	}
	return len(fileTags) == 0
}

// FindGoFunc will locate a function or method within a directory of source files.
// Use "receiver.func" for methods, a function name without the dot for functions.
//
func FindGoFuncIn(funcName string, dir string, tags []string) (file string, err kv.Error) {
	fs := token.NewFileSet()
	pkgs, errGo := parser.ParseDir(fs, dir,
		func(fi os.FileInfo) (isOK bool) {
			return GoFileTags(filepath.Join(dir, fi.Name()), tags)
		}, 0)
	if errGo != nil {
		return file, kv.Wrap(errGo).With("dir", dir).With("stack", stack.Trace().TrimRuntime())
	}

	for _, pkg := range pkgs {
		ast.Inspect(pkg, func(n ast.Node) bool {
			if fun, ok := n.(*ast.FuncDecl); ok {
				name := fun.Name.Name
				if fun.Recv != nil {
					for _, v := range fun.Recv.List {
						switch xv := v.Type.(type) {
						case *ast.StarExpr:
							if si, ok := xv.X.(*ast.Ident); ok {
								name = fmt.Sprintf("%s.%s", si.Name, fun.Name)
							}
						case *ast.Ident:
							name = fmt.Sprintf("%s.%s", xv.Name, fun.Name)
						}
					}
				}
				if name == funcName {
					file = fs.Position(fun.Name.NamePos).Filename
					return false
				}
			}
			return true
		})
	}
	return file, nil
}

// FindGoGenerateFiles is used to descend recursively into a directory to locate go generate
// files
//
func FindGoGenerateFiles(dir string, tags []string) (genFiles []string, err kv.Error) {
	genFiles = []string{}
	foundDirs := map[string]struct{}{}

	dirs := []string{}

	walker := func(path string, info os.FileInfo, err error) error {

		if info == nil || !info.IsDir() {
			return nil
		}

		dirs = append(dirs, path)

		return nil
	}
	if errGo := filepath.Walk(dir, walker); errGo != nil {
		return genFiles, kv.Wrap(errGo).With("dir", dir).With("stack", stack.Trace().TrimRuntime())
	}

	for _, dir := range dirs {
		// Avoid adding duplicates
		if _, isPresent := foundDirs[dir]; isPresent {
			continue
		}
		fs := token.NewFileSet()
		pkgs, errGo := parser.ParseDir(fs, dir,
			func(fi os.FileInfo) (isOK bool) {
				return GoFileTags(filepath.Join(dir, fi.Name()), tags)
			}, parser.ParseComments)
		if errGo != nil {
			return genFiles, kv.Wrap(errGo).With("dir", dir).With("stack", stack.Trace().TrimRuntime())
		}

		func() {
			for _, pkg := range pkgs {
				for fn, f := range pkg.Files {
					for _, comment := range f.Comments {
						pos := fs.PositionFor(comment.Pos(), true)
						if pos.Line == 1 && pos.Column == 1 {
							if strings.HasPrefix(comment.Text(), "go:generate") {
								genFiles = append(genFiles, fn)
							}
						}
					}
				}
			}
		}()
	}

	return genFiles, nil
}

// FindGoGenerateDirs is used to locate and directories that contain
// go files that have well formed fo generate directives.  The directories
func FindGoGenerateDirs(dirs []string, tags []string) (genFiles []string, err kv.Error) {
	genFiles = []string{}
	foundFiles := map[string]struct{}{}
	for _, dir := range dirs {
		files, err := FindGoGenerateFiles(dir, tags)
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			continue
		}
		for _, file := range files {
			// Avoid duplicates
			if _, isPresent := foundFiles[file]; !isPresent {
				foundFiles[file] = struct{}{}
				genFiles = append(genFiles, file)
			}
		}
	}
	return genFiles, nil
}

// FindPossibleGoFuncs can be used to hunt down directories where there was a function found
// that matches one of the specifications of the user, or if as a result of an error during
// checking we might not be sure that the function does not exist
//
func FindPossibleGoFuncs(names []string, dirs []string, tags []string) (possibles []string, err kv.Error) {
	possibles = []string{}
	files := map[string]struct{}{}
	for _, dir := range dirs {
		for _, name := range names {
			// Some what inefficent as we are scanning the dir
			// potentially multiple times
			file, err := FindGoFuncIn(name, dir, tags)
			if err == nil && len(file) == 0 {
				continue
			}
			// Avoid duplicates
			if _, isPresent := files[file]; !isPresent {
				files[file] = struct{}{}
				possibles = append(possibles, file)
			}
		}
	}
	return possibles, nil
}

func (md *MetaData) GoBuild(tags []string, opts []string, outputDir string, outputSuffix string, versionBump bool) (outputs []string, err kv.Error) {

	// Dont do any version manipulation if we are just preparing images
	// As we begin the build determine if we are using a pre-released version
	// and if so automatically bump the pre-release version to reflect a development
	// step
	if versionBump {
		if len(md.SemVer.Prerelease()) != 0 {
			if _, err = md.BumpPrerelease(); err != nil {
				return outputs, err
			}
		}
	}

	if outputs, err = md.GoSimpleBuild(tags, opts, outputDir, outputSuffix); err != nil {
		return []string{}, err
	}

	return outputs, nil
}

func runCMD(cmds []string, logOut io.Writer, logErr io.Writer) (err kv.Error) {

	cmd := exec.Command("bash", "-c", strings.Join(cmds, " && "))
	cmd.Stdout = logOut
	cmd.Stderr = logErr

	if errGo := cmd.Start(); errGo != nil {
		dir, _ := os.Getwd()
		fmt.Fprintln(os.Stderr, kv.Wrap(errGo, "unable to run the compiler").
			With("stack", stack.Trace().TrimRuntime()).With("cmds", strings.Join(cmds, "¶ ")).
			With("dir", dir).Error())
		os.Exit(-3)
	}

	if errGo := cmd.Wait(); errGo != nil {
		return kv.Wrap(errGo, "unable to run the compiler").
			With("stack", stack.Trace().TrimRuntime()).With("cmds", strings.Join(cmds, "¶ "))
	}
	return nil
}

// GoGenerate will invoke the go generator.  This method must resort to using the
// command line exec as there is no public library within the go code base that
// exposes the go generator
//
func (md *MetaData) GoGenerate(file string, env map[string]string, tags []string, opts []string) (outputs []string, err kv.Error) {
	outputs = []string{}

	buildEnv := make([]string, 0, len(env))

	for k, v := range env {
		buildEnv = append(buildEnv, fmt.Sprintf("%s='%s'", k, v))
	}

	tagOption := ""
	if len(tags) > 0 {
		tagOption = fmt.Sprintf(" -tags \"%s\" ", strings.Join(tags, " "))
	}

	goPath := os.Getenv("GOPATH")
	cmds := []string{
		fmt.Sprintf("%s/bin/dep ensure || true", goPath),
		fmt.Sprintf("%s go generate %s %s %s",
			strings.Join(buildEnv, " "), file, strings.Join(opts, " "), tagOption),
	}

	outBuf := &strings.Builder{}
	if err = runCMD(cmds, io.MultiWriter(os.Stdout, outBuf), io.MultiWriter(os.Stderr, outBuf)); err != nil {
		outputs = append(outputs, strings.Split(outBuf.String(), "\n")...)
		return outputs, err.With("module", md.Module).With("file", file)
	}
	return outputs, nil
}

func (md *MetaData) GoSimpleBuild(tags []string, opts []string, outputDir string, outputSuffix string) (outputs []string, err kv.Error) {
	outputs = []string{}

	// Copy the compiled file into the GOPATH bin directory
	goPath := os.Getenv("GOPATH")
	if len(goPath) == 0 {
		return outputs, kv.NewError("unable to determine the compiler bin output dir, env var GOPATH might be missing or empty").With("stack", stack.Trace().TrimRuntime())
	}

	if len(outputDir) == 0 {
		outputDir = "./bin"
	}

	if outputs, err = md.GoCompile(map[string]string{}, tags, opts, outputDir, outputSuffix); err != nil {
		return outputs, err
	}

	// Any executable binaries are copied into your $GOPATH/bin automatically
	if errGo := os.MkdirAll(filepath.Join(goPath, "bin"), os.ModePerm); errGo != nil {
		if !os.IsExist(errGo) {
			return outputs, kv.Wrap(errGo, "unable to create the $GOPATH/bin directory").With("stack", stack.Trace().TrimRuntime())
		}
	}

	// Find any executables we have and copy them to the gopath bin directory as well
	binPath, errGo := filepath.Abs(filepath.Join(".", "bin"))
	if errGo != nil {
		return outputs, kv.Wrap(errGo, "unable to copy binary files from the ./bin directory").With("stack", stack.Trace().TrimRuntime())
	}

	errGo = filepath.Walk(binPath, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		// Is the file executable at all ?
		if f.Mode()&0111 != 0 {
			src := filepath.Join("bin", f.Name())
			dst := filepath.Join(goPath, "bin", filepath.Base(f.Name()))

			if err := CopyFile(src, dst); err != nil {
				return err
			}
			outputs = append(outputs, dst)
		}
		return nil
	})

	if errGo == nil {
		return outputs, nil
	}

	return outputs, errGo.(kv.Error)
}

func (md *MetaData) GoFetchBuilt() (outputs []string, err kv.Error) {
	outputs = []string{}

	binPath, errGo := filepath.Abs(filepath.Join(".", "bin"))
	if errGo != nil {
		return outputs, kv.Wrap(errGo, "unable to find binary files").With("stack", stack.Trace().TrimRuntime())
	}

	// Nothing to be found which is a valid condition
	if fi, err := os.Stat(binPath); err != nil || !fi.IsDir() {
		return outputs, nil
	}

	errGo = filepath.Walk(binPath, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		// Is the file executable at all ?
		if f.Mode()&0111 != 0 {
			outputs = append(outputs, filepath.Join(binPath, f.Name()))
		}
		return nil
	})

	if errGo == nil {
		return outputs, nil
	}

	return outputs, errGo.(kv.Error)
}

func (md *MetaData) GoCompile(env map[string]string, tags []string, opts []string, outputDir string, outputSuffix string) (outputs []string, err kv.Error) {
	if errGo := os.Mkdir("bin", os.ModePerm); errGo != nil {
		if !os.IsExist(errGo) {
			return outputs, kv.Wrap(errGo, "unable to create the bin directory").With("stack", stack.Trace().TrimRuntime())
		}
	}

	// prepare flags and options needed for the actual build
	ldFlags := []string{}
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/karlmutch/duat/version.BuildTime=%s", time.Now().Format("2006-01-02_15:04:04-0700")))
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/karlmutch/duat/version.GitHash=%s", md.Git.Hash))
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/karlmutch/duat/version.SemVer=\"%s\"", md.SemVer.String()))

	buildOS, hasOS := os.LookupEnv("GOOS")
	arch, hasArch := os.LookupEnv("GOARCH")

	if !hasOS {
		buildOS = runtime.GOOS
	}
	if !hasArch {
		arch = runtime.GOARCH
	}
	if arch == "arm" {
		if arm, isPresent := os.LookupEnv("GOARM"); isPresent {
			arch += arm
		}
	}

	buildEnv := []string{"GO_ENABLED=0"}
	// If the LD_LIBRARY_PATH is present bring it into the build automatically
	if ldPath, hasLdPath := os.LookupEnv("LD_LIBRARY_PATH"); hasLdPath {
		buildEnv = append(buildEnv, fmt.Sprintf("LD_LIBRARY_PATH=%s", ldPath))
	}

	for k, v := range env {
		buildEnv = append(buildEnv, fmt.Sprintf("%s='%s'", k, v))
		switch k {
		case "GOOS":
			buildOS = v
		case "GOARCH":
			arch = v
		}
	}

	goPath, isPresent := os.LookupEnv("GOPATH")
	if !isPresent {
		goPath, _ = os.LookupEnv("HOME")
	}
	trimpath := ""
	if len(goPath) != 0 {
		trimpath = "-gcflags \"all=-trimpath=" + goPath + "\""
	}

	output := md.Module + "-" + buildOS + "-" + arch
	if len(outputSuffix) != 0 {
		output += "-" + outputSuffix
	}
	output = filepath.Join(outputDir, output)

	tagOption := ""
	if len(tags) > 0 {
		tagOption = fmt.Sprintf(" -tags \"%s\" ", strings.Join(tags, " "))
	}

	cmds := []string{
		fmt.Sprintf("%s/bin/dep ensure || true", goPath),
		fmt.Sprintf(("%s go build %s %s %s -ldflags \"" + strings.Join(ldFlags, " ") + "\" -o " + output + " ."),
			strings.Join(buildEnv, " "), strings.Join(opts, " "), trimpath, tagOption),
	}

	outBuf := &strings.Builder{}
	if err = runCMD(cmds, io.MultiWriter(os.Stdout, outBuf), io.MultiWriter(os.Stderr, outBuf)); err != nil {
		outputs = append(outputs, strings.Split(outBuf.String(), "\n")...)
		return outputs, err.With("module", md.Module)
	}
	return outputs, nil
}

func (md *MetaData) GoTest(env map[string]string, tags []string, opts []string) (err kv.Error) {

	// prepare flags and options needed for the actual build
	ldFlags := []string{}
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/karlmutch/duat/version.BuildTime=%s", time.Now().Format("2006-01-02_15:04:04-0700")))
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/karlmutch/duat/version.GitHash=%s", md.Git.Hash))
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/karlmutch/duat/version.SemVer=\"%s\"", md.SemVer.String()))

	buildEnv := []string{"GO_ENABLED=0"}
	// If the LD_LIBRARY_PATH is present bring it into the build automatically
	if ldPath, hasLdPath := os.LookupEnv("LD_LIBRARY_PATH"); hasLdPath {
		buildEnv = append(buildEnv, fmt.Sprintf("LD_LIBRARY_PATH=%s", ldPath))
	}

	for k, v := range env {
		buildEnv = append(buildEnv, fmt.Sprintf("%s='%s'", k, v))
	}

	tagOption := ""
	if len(tags) > 0 {
		tagOption = fmt.Sprintf(" -tags \"%s\" ", strings.Join(tags, " "))
	}

	goPath := os.Getenv("GOPATH")
	cmds := []string{
		fmt.Sprintf("%s/bin/dep ensure || true", goPath),
		fmt.Sprintf(("%s go test %s -ldflags \"" + strings.Join(ldFlags, " ") + "\" %s ."),
			strings.Join(buildEnv, " "), tagOption, strings.Join(opts, " ")),
	}

	cmd := exec.Command("bash", "-c", strings.Join(cmds, " && "))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if errGo := cmd.Start(); errGo != nil {
		dir, _ := os.Getwd()
		fmt.Fprintln(os.Stderr, kv.Wrap(errGo, "unable to run the test").With("module", md.Module).
			With("stack", stack.Trace().TrimRuntime()).With("cmds", strings.Join(cmds, "¶ ")).
			With("dir", dir).Error())
		os.Exit(-3)
	}

	if errGo := cmd.Wait(); errGo != nil {
		return kv.Wrap(errGo, "unable to run the compiler").
			With("stack", stack.Trace().TrimRuntime()).With("cmds", strings.Join(cmds, "¶ "))
	}
	return nil
}
