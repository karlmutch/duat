package duat

// Contains the implementation of an image build function

import (
	"fmt"
	"os/user"
	"path/filepath"

	"github.com/karlmutch/duat/internal/image"
	"github.com/karlmutch/errors"
	"github.com/karlmutch/stack"
)

var (
	usrData = &user.User{}
	usrErr  = errors.New("")

	stateDir = ""
)

func init() {
	usr, errGo := user.Current()
	if errGo != nil {
		usrErr = errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		return
	}
	usrErr = nil

	usrData = usr
	stateDir = filepath.Join(usr.HomeDir, ".local", "share", "img")
}

func (md *MetaData) ImageBuild(dir string, target string, tag string, buildArgs []string, output string) (err errors.Error) {

	if usrErr != nil {
		return usrErr
	}

	args := append([]string{}, buildArgs...)
	args = append(args, fmt.Sprintf("USER=%s", usrData.Username))
	args = append(args, fmt.Sprintf("USER_ID=%s", usrData.Uid))
	args = append(args, fmt.Sprintf("USER_GROUP_ID=%s", usrData.Gid))

	image, err := image.NewBuildCmd(dir, target, tag, args)
	if err != nil {
		return err
	}

	logger := make(chan string)
	defer close(logger)

	go func() {
		for {
			if logger == nil {
				return
			}
			select {
			case msg := <-logger:
				if len(msg) == 0 {
					return
				}
			}
		}
	}()

	if err = image.Build(stateDir, logger); err != nil {
		return err
	}

	// Save the image

	// Remove the working image, but dont prune that is for another function
	return nil
}

func (md *MetaData) ImagePrune() (err errors.Error) {

	if usrErr != nil {
		return usrErr
	}

	return image.Prune(stateDir)
}
