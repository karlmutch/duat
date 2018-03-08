package duat

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/karlmutch/amicontained/container" // Forked from https://github.com/jessfraz/amicontained.git

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

var (
	ErrInContainer = errors.New("operation not support while within a running container")
)

// ScrubRepository is used to transform module names into docker image name compliant strings.
// The definition of a repository name string can be found at https://github.com/moby/moby/blob/master/image/spec/v1.2.md
//
func (*MetaData) ScrubDockerRepo(name string) (repoName string, err errors.Error) {
	for _, aChar := range name {
		if aChar < '0' || aChar > 'z' || (aChar > '9' && aChar < 'A') || (aChar > 'Z' && aChar < 'a') {
			if aChar != '.' && aChar != '_' && aChar != '-' {
				repoName += "-"
				continue
			}
		}
		repoName += string(aChar)
	}

	return strings.ToLower(repoName), nil
}

// IsDocker is used to determine if we are running within a container runtime
func (*MetaData) ContainerRuntime() (containerType string, err errors.Error) {
	name, errGo := container.DetectRuntime()
	if errGo != nil {
		if errGo == container.ErrContainerRuntimeNotFound {
			return "", nil
		}
		return "", errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	if name == "not-found" {
		return "", nil
	}
	return name, nil
}

func (md *MetaData) GenerateImageName() (repo string, version string, prerelease bool, err errors.Error) {

	// Get the git repo name and the parent which will have been used to name the
	// containers being created during our build process
	gitParts := strings.Split(md.Git.URL, "/")
	label := strings.TrimSuffix(gitParts[len(gitParts)-1], ".git")

	// Look for pre-release components within the version string
	preParts := strings.Split(md.SemVer.Prerelease(), "-")

	return fmt.Sprintf("%s/%s/%s", gitParts[len(gitParts)-2], label, md.Module), md.SemVer.String(), len(preParts) >= 2, nil
}

// ImagePrune will wipe the images from the local registry, the all option can be used to leave the latest image present
//
func (md *MetaData) ImagePrune(all bool) (err errors.Error) {

	repo, ver, pre, err := md.GenerateImageName()
	if err != nil {
		return err
	}
	tag := repo
	tag += ":"
	tag += ver

	if pre {
		// Drop the time stamp portion of the prerelease
		parts := strings.Split(tag, "-")
		tag = strings.Join(parts[:len(parts)-2], "-")
	}

	// Now get our local docker images
	dock, errGo := client.NewEnvClient()
	if errGo != nil {
		return errors.Wrap(errGo, "docker unavailable").With("stack", stack.Trace().TrimRuntime())
	}

	images, errGo := dock.ImageList(context.Background(), types.ImageListOptions{})
	if errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	// filter the image repo and version tags into a collection of strings that can
	// be sorted using their pre-release tag so that we can groom them in a LRU
	// fashion
	//
	knownTags := make([]string, 0, len(images))
	knownIDs := make(map[string]string, len(images))

	for _, image := range images {
		for _, repo := range image.RepoTags {
			if strings.HasPrefix(repo, tag) {
				knownTags = append(knownTags, repo)
				knownIDs[repo] = image.ID
			}
		}
	}

	sort.Strings(knownTags)

	if len(knownTags) != 0 {
		if !all {
			knownTags = knownTags[:len(knownTags)-1]
		}
	}

	// Now we have a list of the known tags that are for images we wish to remove
	for _, tag := range knownTags {
		image, isPresent := knownIDs[tag]
		if isPresent {
			if _, errGo = dock.ImageRemove(context.Background(), image, types.ImageRemoveOptions{}); errGo != nil {
				return errors.Wrap(errGo).With("imageID").With("stack", stack.Trace().TrimRuntime())
			}
			delete(knownIDs, tag)
		}
	}
	return nil
}

func (md *MetaData) ImageExists() (exists bool, err errors.Error) {
	runtime, err := md.ContainerRuntime()
	if err != nil {
		return false, err
	}
	if len(runtime) > 0 {
		return false, errors.Wrap(ErrInContainer).With("stack", stack.Trace().TrimRuntime())
	}

	repoName, tag, _, err := md.GenerateImageName()
	if err != nil {
		return false, err
	}
	fullTag := fmt.Sprintf("%s:%s", repoName, tag)

	// Now get our local docker images
	dock, errGo := client.NewEnvClient()
	if errGo != nil {
		return false, errors.Wrap(errGo, "docker unavailable").With("stack", stack.Trace().TrimRuntime())
	}

	images, errGo := dock.ImageList(context.Background(), types.ImageListOptions{})
	if errGo != nil {
		return false, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	for _, image := range images {
		for _, repo := range image.RepoTags {
			if repo == fullTag {
				return true, nil
			}
		}
	}

	return false, nil
}

func (md *MetaData) ImageCreate() (err errors.Error) {

	runtime, err := md.ContainerRuntime()
	if err != nil {
		return err
	}
	if len(runtime) > 0 {
		return errors.Wrap(ErrInContainer).With("stack", stack.Trace().TrimRuntime())
	}

	// Now prepare a temporary Dockerfile that has been processed using the semver injection
	fn, errGo := ioutil.TempFile(".", "Dockerfile.")
	if errGo != nil {
		return errors.Wrap(errGo, "a temporary Dockerfile could not be created in your working directory").With("stack", stack.Trace().TrimRuntime())
	}
	fn.Close()
	defer os.Remove(fn.Name())

	if err = md.Replace("./Dockerfile", fn.Name(), true); err != nil {
		return errors.Wrap(errGo, "a temporary Dockerfile was not generated").With("tempFile", fn).With("stack", stack.Trace().TrimRuntime())
	}

	repoName, tag, _, err := md.GenerateImageName()
	if err != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	user := os.Getenv("USER")
	if len(user) == 0 {
		return errors.New("could not retrieve user ID from env vars").With("stack", stack.Trace().TrimRuntime())
	}

	buildArgs := fmt.Sprintf("--build-arg USER=%s --build-arg USER_ID=`id -u %s` --build-arg USER_GROUP_ID=`id -g %s`", user, user, user)
	cmds := []string{
		fmt.Sprintf("docker build -t %s:%s %s -f %s .", repoName, tag, buildArgs, fn.Name()),
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

	// Cull out any older dangling images left from previous runs, the false
	// is to indicate that at least one image should remain
	if err = md.ImagePrune(false); err != nil {
		return err
	}
	return nil
}
