package duat

// This file contains methods for Go builds using the duat conventions

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go/ast"
	"go/parser"
	"go/token"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

var (
	goPath = os.Getenv("GOPATH")
)

// Look for directories inside the root 'dir' and return their paths, skip any vendor directories
//
func findDirs(dir string) (dirs []string, err errors.Error) {
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
		return nil, errors.Wrap(errGo).With("dir", dir).With("stack", stack.Trace().TrimRuntime())
	}
	return dirs, err
}

func FindGoDirs(dir string) (dirs []string, err errors.Error) {
	dirs = []string{}

	found, err := findDirs(dir)
	if err != nil {
		return []string{}, err
	}

	groomed, err := FindPossibleGoFunc("main", found, []string{})
	if err != nil {
		return []string{}, err
	}

	for _, dir := range groomed {
		dirs = append(dirs, filepath.Dir(dir))
	}
	return dirs, nil
}

func FindGoFiles(dir string) (files []string, err errors.Error) {
	files = []string{}

	if stat, errGo := os.Stat(dir); errGo != nil {
		return files, errors.Wrap(errGo).With("dir", dir).With("stack", stack.Trace().TrimRuntime())
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
	for scan.Scan() {
		tokens := strings.Split(scan.Text(), " ")
		if tokens[0] != "//" {
			break
		}
		fileTags := map[string]struct{}{}
		for _, token := range tokens[1:] {
			if len(token) != 0 {
				fileTags[token] = struct{}{}
			}
		}
		if _, isPresent := fileTags["+build"]; !isPresent {
			break
		}
		delete(fileTags, "+build")
		for _, aTag := range tags {
			delete(fileTags, aTag)
		}
		return len(fileTags) == 0
	}
	return true
}

// FindGoFunc will locate a function or method within a directory of source files.
// Use "receiever.func" for methods, a function name without the dot for functions.
//
func FindGoFuncIn(funcName string, dir string, tags []string) (file string, err errors.Error) {
	fs := token.NewFileSet()
	pkgs, errGo := parser.ParseDir(fs, dir,
		func(fi os.FileInfo) (isOK bool) {
			return GoFileTags(filepath.Join(dir, fi.Name()), tags)
		}, 0)
	if errGo != nil {
		return file, errors.Wrap(errGo).With("dir", dir).With("stack", stack.Trace().TrimRuntime())
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

// FindPossibleGoFunc can be used to hunt down directories where there was a function found
// that matches the specification of the user, or if as a result of an error during
// checking we might not be sure that the function does not exist
//
func FindPossibleGoFunc(name string, dirs []string, tags []string) (possibles []string, err errors.Error) {
	possibles = []string{}
	for _, dir := range dirs {
		file, err := FindGoFuncIn(name, dir, tags)
		if err == nil && len(file) == 0 {
			continue
		}
		possibles = append(possibles, file)
	}
	return possibles, nil
}

func (md *MetaData) GoDockerBuild(imageOnly bool, prune bool) (outputs []string, err errors.Error) {

	// Dont do any version manipulation if we are just preparing images
	if !imageOnly {
		// As we begin the build determine if we are using a pre-released version
		// and if so automatically bump the pre-release version to reflect a development
		// step
		if len(md.SemVer.Prerelease()) != 0 {
			if _, err = md.BumpPrerelease(); err != nil {
				return outputs, err
			}
		}
	}

	// If there is a Dockerfile for this module then check the images etc
	image := false
	if _, err := os.Stat("./Dockerfile"); err == nil {
		if runtime, _ := md.ContainerRuntime(); len(runtime) == 0 {
			exists, _, err := md.ImageExists()
			if err != nil {
				return outputs, err
			}
			if exists {
				return outputs, errors.New("an image already exists at the current software version, using 'semver pre' to bump your pre-release version will correct this").With("stack", stack.Trace().TrimRuntime())
			}
		}
		image = true
	}

	if !imageOnly {
		if outputs, err = md.GoBuild(); err != nil {
			return []string{}, err
		}
		// If there is a Dockerfile indicating that the release product is an image then we dont
		// include any go binaries created as outputs as the Docker image consumes them
		if image {
			outputs = []string{}
		}
	}

	// If we have a Dockerfile in our target directory build it, unless we are running in a container then dont
	if runtime, _ := md.ContainerRuntime(); len(runtime) == 0 {
		if _, err := os.Stat("Dockerfile"); err == nil {
			// Create an image
			logged := strings.Builder{}
			if err := md.ImageCreate(ioutil.Discard); err != nil {
				if errors.Cause(err) == ErrInContainer {
					// This only a real error if the user explicitly asked for the image to be produced
					if imageOnly {
						return outputs, errors.New("-image-only used but we were running inside a container which is not supported").With("stack", stack.Trace().TrimRuntime())
					}
				} else {
					fmt.Fprint(os.Stderr, logged.String())
					return []string{}, err
				}
			}
			if prune {
				if err := md.ImagePrune(false); err != nil {
					fmt.Fprintln(os.Stderr, err.With("msg", "prune operation failed, and ignored").Error())
				}
			}
		} else {
			if imageOnly {
				return outputs, errors.New("-image-only used however there is no Dockerfile present").With("stack", stack.Trace().TrimRuntime())
			}
		}
	}
	return outputs, nil
}

func (md *MetaData) GoBuild() (outputs []string, err errors.Error) {
	outputs = []string{}

	// Copy the compiled file into the GOPATH bin directory
	if len(goPath) == 0 {
		return outputs, errors.New("unable to determine the compiler bin output dir, env var GOPATH might be missing or empty").With("stack", stack.Trace().TrimRuntime())
	}

	if err = md.GoCompile(); err != nil {
		return outputs, err
	}

	if errGo := os.MkdirAll(filepath.Join(goPath, "bin"), os.ModePerm); errGo != nil {
		if !os.IsExist(errGo) {
			return outputs, errors.Wrap(errGo, "unable to create the $GOPATH/bin directory").With("stack", stack.Trace().TrimRuntime())
		}
	}

	// Find any executables we have and copy them to the gopath bin directory as well
	binPath, errGo := filepath.Abs(filepath.Join(".", "bin"))
	if errGo != nil {
		return outputs, errors.Wrap(errGo, "unable to copy binary files from the ./bin directory").With("stack", stack.Trace().TrimRuntime())
	}

	errGo = filepath.Walk(binPath, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		// Is the file executable at all ?
		if f.Mode()&0111 != 0 {
			src := filepath.Join("bin", f.Name())
			dst := filepath.Join(goPath, "bin", filepath.Base(f.Name()))

			if err = CopyFile(src, dst); err != nil {
				return err
			}
			outputs = append(outputs, dst)
		}
		return nil
	})

	if errGo == nil {
		return outputs, nil
	}

	return outputs, errGo.(errors.Error)
}

func (md *MetaData) GoFetchBuilt() (outputs []string, err errors.Error) {
	outputs = []string{}

	binPath, errGo := filepath.Abs(filepath.Join(".", "bin"))
	if errGo != nil {
		return outputs, errors.Wrap(errGo, "unable to find binary files").With("bin", binPath).With("stack", stack.Trace().TrimRuntime())
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

	return outputs, errGo.(errors.Error)
}

func (md *MetaData) GoCompile() (err errors.Error) {
	if errGo := os.Mkdir("bin", os.ModePerm); errGo != nil {
		if !os.IsExist(errGo) {
			return errors.Wrap(errGo, "unable to create the bin directory").With("stack", stack.Trace().TrimRuntime())
		}
	}

	// prepare flags and options needed for the actual build
	ldFlags := []string{}
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/karlmutch/duat/version.BuildTime=%s", time.Now().Format("2006-01-02_15:04:04-0700")))
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/karlmutch/duat/version.GitHash=%s", md.Git.Hash))
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/karlmutch/duat/version.SemVer=\"%s\"", md.SemVer.String()))

	cmds := []string{
		fmt.Sprintf("%s/bin/dep ensure", goPath),
		fmt.Sprintf(("GO_ENABLED=0 go build -ldflags \"" + strings.Join(ldFlags, " ") + "\" -o bin/" + md.Module + " .\n")),
	}

	cmd := exec.Command("bash", "-c", strings.Join(cmds, " && "))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if errGo := cmd.Start(); errGo != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(errGo, "unable to run the compiler").With("module", md.Module).With("stack", stack.Trace().TrimRuntime()).Error())
		os.Exit(-3)
	}

	if errGo := cmd.Wait(); errGo != nil {
		return errors.Wrap(errGo, "unable to run the compiler").With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}
