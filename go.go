package duat

// This file contains methods for Go builds using the duat conventions

import (
	"fmt"
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

// FindFunc will locate a function or method within a directory of source files.
// Use "receiever.func" for methods, a function name without the dot for functions.
//
func FindFuncIn(funcName string, dir string) (file string, err errors.Error) {
	fs := token.NewFileSet()
	pkgs, errGo := parser.ParseDir(fs, dir, nil, 0)
	if errGo != nil {
		return "", errors.Wrap(errGo).With("dir", dir).With("stack", stack.Trace().TrimRuntime())
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

// FindPossibleFunc can be used to hunt down directories where there was a function found
// that matches the specification of the user, or if as a result of an error during
// checking we might not be sure that the function does not exist
//
func FindPossibleFunc(name string, dirs []string) (possibles []string, err errors.Error) {
	possibles = []string{}
	for _, dir := range dirs {
		file, err := FindFuncIn(name, dir)
		if err == nil && len(file) == 0 {
			continue
		}
		possibles = append(possibles, file)
	}
	return possibles, nil
}

func (md *MetaData) GoBuild() (err errors.Error) {
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
		}
		return nil
	})
	if errGo == nil {
		return nil
	}

	return errGo.(errors.Error)
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
		fmt.Fprintln(os.Stderr, errors.Wrap(errGo, "unable to run the compiler").With("stack", stack.Trace().TrimRuntime()).Error())
		os.Exit(-3)
	}

	if errGo := cmd.Wait(); errGo != nil {
		return errors.Wrap(errGo, "unable to run the compiler").With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}
