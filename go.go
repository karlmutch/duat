package duat

// This file contains methods for Go builds using the duat conventions

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

var (
	goPath = os.Getenv("GOPATH")
)

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
