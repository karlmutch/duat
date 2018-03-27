package duat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"
	"unicode"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"

	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks
	"github.com/karlmutch/semver" // Forked copy of https://github.com/Masterminds/semver

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

// ReleaseImage(repo string)
//
// Use GenerateImageName
// See if an existing release is locally present, which is OK but make sure the id is the same.
// Tag the pre-release with the clean release, if requested using erasePrerelease.
// Prepend and push
func (md *MetaData) ImageRelease(remote string, erasePrerelease bool) (images []string, err errors.Error) {

	images = []string{}

	if len(remote) != 0 && !strings.HasSuffix(remote, ".amazonaws.com") {
		return images, errors.New("an external repo was specified but was not recognized as being from AWS").With("repo", remote).With("stack", stack.Trace().TrimRuntime())
	}

	// Make sure the original image exists for the current version before processing
	// a release version
	exists, id, err := md.ImageExists()
	if err != nil {
		return images, err
	}
	if !exists {
		return images, errors.New("a pre-release image must exist before calling release").With("stack", stack.Trace().TrimRuntime())
	}

	curRepo, curVersion, _, err := md.GenerateImageName()
	if err != nil {
		return images, err
	}
	curTag := fmt.Sprintf("%s:%s", curRepo, curVersion)

	repo, version, _, err := md.generateImageName(md.SemVer)
	if err != nil {
		return images, err
	}
	newVersion, errGo := semver.NewVersion(version)
	if errGo != nil {
		return images, errors.Wrap(errGo, "could not parse the new release").With("version", version).With("stack", stack.Trace().TrimRuntime())
	}
	exists, relID, err := md.imageExists(newVersion)
	if err != nil {
		return images, err
	}
	if exists {
		if relID != id {
			return images, errors.New("the released image already exists, with a different image ID").With("stack", stack.Trace().TrimRuntime())
		}
	}

	dock, errGo := client.NewEnvClient()
	if errGo != nil {
		return images, errors.Wrap(errGo, "docker unavailable").With("stack", stack.Trace().TrimRuntime())
	}
	tag := fmt.Sprintf("%s:%s", repo, version)
	if !exists {
		// Tag the pre-released version
		// Now get our local docker images

		errGo = dock.ImageTag(context.Background(), curTag, tag)
		if errGo != nil {
			return images, errors.Wrap(errGo).With("curTag", curTag).With("newTag", tag).With("stack", stack.Trace().TrimRuntime())
		}
	}
	images = append(images, tag)

	if len(remote) != 0 {
		if err = CreateECRRepo(repo); err != nil {

			if aerr, ok := errors.Cause(err).(awserr.Error); ok {
				switch aerr.Code() {
				case ecr.ErrCodeRepositoryAlreadyExistsException:
					break
				default:
					return []string{}, errors.Wrap(err).With("curTag", curTag).With("newTag", tag).With("stack", stack.Trace().TrimRuntime())
				}
			} else {
				return []string{}, errors.Wrap(err).With("curTag", curTag).With("newTag", tag).With("stack", stack.Trace().TrimRuntime())
			}
		}

		tag = fmt.Sprintf("%s/%s:%s", remote, repo, version)
		if errGo := dock.ImageTag(context.Background(), curTag, tag); errGo != nil {
			return images, errors.Wrap(errGo).With("curTag", curTag).With("newTag", tag).With("stack", stack.Trace().TrimRuntime())
		}

		token, err := GetECRToken()
		if err != nil {
			return images, err.With("newTag", tag).With("stack", stack.Trace().TrimRuntime())
		}
		// For a full explanation of the reformating of the auth string please see,
		// https://github.com/moby/moby/issues/33552
		authInfoBytes, _ := base64.StdEncoding.DecodeString(token)
		authInfo := strings.Split(string(authInfoBytes), ":")
		auth := struct {
			Username string
			Password string
		}{
			Username: authInfo[0],
			Password: authInfo[1],
		}

		authBytes, _ := json.Marshal(auth)
		token = base64.StdEncoding.EncodeToString(authBytes)

		pushOptions := types.ImagePushOptions{
			RegistryAuth: token,
		}

		resp, errGo := dock.ImagePush(context.Background(), tag, pushOptions)
		if errGo != nil {
			return images, errors.Wrap(errGo).With("curTag", curTag).With("newTag", tag).With("stack", stack.Trace().TrimRuntime())
		}
		if _, errGo = ioutil.ReadAll(resp); errGo != nil {
			return images, errors.Wrap(errGo).With("curTag", curTag).With("newTag", tag).With("stack", stack.Trace().TrimRuntime())
		}
		images = append(images, tag)
	}
	return images, nil
}

func (md *MetaData) generateImageName(semVer *semver.Version) (repo string, version string, prerelease bool, err errors.Error) {
	// Get the git repo name and the parent which will have been used to name the
	// containers being created during our build process
	gitParts := strings.Split(md.Git.URL.EscapedPath(), "/")

	label := gitParts[len(gitParts)-1]
	if strings.HasSuffix(label, ".git") {
		label = strings.TrimSuffix(label, ".git")
	}

	// Look for pre-release components within the version string
	preParts := strings.Split(semVer.Prerelease(), "-")

	repoName := fmt.Sprintf("%s/%s/%s", gitParts[len(gitParts)-2], label, md.Module)
	// docker repositories only use lowercase _ and -
	dockerRepo := strings.Builder{}
	for i, char := range repoName {
		if unicode.IsUpper(char) && i != 0 {
			dockerRepo.WriteRune('-')
		}
		dockerRepo.WriteRune(unicode.ToLower(char))
	}
	return dockerRepo.String(), semVer.String(), len(preParts) >= 2, nil
}

// caller should
// Clean the semver and save the new one after the push

func (md *MetaData) GenerateImageName() (repo string, version string, prerelease bool, err errors.Error) {
	return md.generateImageName(md.SemVer)
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

	switch len(knownTags) {
	case 0:
		return nil
	case 1:
		if !all {
			return nil
		}
	default:
		// We have more than one image so save the latest image, and its associated tags
		sort.Strings(knownTags)

		if !all {
			// When keep an image find all repos that have that image and remove then
			// from the manifest so that we dont destroy what we wish to keep
			repo := knownTags[len(knownTags)-1]
			keepID := knownIDs[repo]
			for k, v := range knownIDs {
				if v == keepID {
					delete(knownIDs, k)
				}
			}
		}
	}

	// Now we have a list of the images we wish to remove, while deleting them there
	// could well be duplicates so keep a record of anything removed and dont repeat
	// operations that would result in errors
	//
	removed := make(map[string]bool, len(knownIDs))
	for _, image := range knownIDs {
		if _, isPresent := removed[image]; isPresent {
			continue
		}
		if _, errGo = dock.ImageRemove(context.Background(), image, types.ImageRemoveOptions{Force: true}); errGo != nil {
			return errors.Wrap(errGo).With("imageID").With("stack", stack.Trace().TrimRuntime())
		}
		removed[image] = true
	}
	return nil
}

func (md *MetaData) imageExists(ver *semver.Version) (exists bool, id string, err errors.Error) {
	runtime, err := md.ContainerRuntime()
	if err != nil {
		return false, "", err
	}
	if len(runtime) > 0 {
		return false, "", errors.Wrap(ErrInContainer).With("stack", stack.Trace().TrimRuntime())
	}

	repoName, tag, _, err := md.generateImageName(ver)
	if err != nil {
		return false, "", err
	}
	fullTag := fmt.Sprintf("%s:%s", repoName, tag)

	// Now get our local docker images
	dock, errGo := client.NewEnvClient()
	if errGo != nil {
		return false, "", errors.Wrap(errGo, "docker unavailable").With("stack", stack.Trace().TrimRuntime())
	}

	images, errGo := dock.ImageList(context.Background(), types.ImageListOptions{})
	if errGo != nil {
		return false, "", errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	for _, image := range images {
		for _, repo := range image.RepoTags {
			if repo == fullTag {
				return true, image.ID, nil
			}
		}
	}

	return false, "", nil
}

func (md *MetaData) ImageExists() (exists bool, id string, err errors.Error) {
	return md.imageExists(md.SemVer)
}

func (md *MetaData) ImageCreate(out io.Writer) (err errors.Error) {

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
	cmd.Stdout = out
	cmd.Stderr = out

	if errGo := cmd.Start(); errGo != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(errGo, "unable to run the compiler").With("stack", stack.Trace().TrimRuntime()).Error())
		os.Exit(-3)
	}

	if errGo := cmd.Wait(); errGo != nil {
		return errors.Wrap(errGo, "unable to run the compiler").With("stack", stack.Trace().TrimRuntime())
	}

	return nil
}
