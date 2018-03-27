package duat

// This file contains functions that are useful for interaction with github to perform operations
// not covered by the core git APIs including things such as release management etc

// Some of the code in this module was snagged and inspired by code from https://github.com/c4milo/github-release
// which is licensed using the Mozilla Public Licence 2.0, https://github.com/c4milo/github-release/blob/master/LICENSE
//
// This file is therefore licensed under the same terms while the larger body of work this
// comes as part of 'duat', might be licensed under similar but different licenses.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

// Release represents a Github Release.
type gitRelease struct {
	UploadURL  string `json:"upload_url,omitempty"`
	TagName    string `json:"tag_name"`
	Branch     string `json:"target_commitish"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	token      string
}

func fileSize(file *os.File) (size int64, err errors.Error) {
	stat, errGo := file.Stat()
	if errGo != nil {
		return 0, errors.Wrap(errGo, "file could not be checked for its size").With("file", file.Name()).With("stack", stack.Trace().TrimRuntime())
	}
	return stat.Size(), nil
}

func (git *gitRelease) githubUpload(url string, path string) (resp string, err errors.Error) {

	file, errGo := os.Open(path)
	if errGo != nil {
		return "", errors.Wrap(errGo, "file does not exist").With("file", path).With("stack", stack.Trace().TrimRuntime())
	}
	defer file.Close()

	size, err := fileSize(file)
	if err != nil {
		return "", err
	}

	rqst := url + "?name=" + filepath.Base(file.Name())
	body, err := doGitRequest("POST", rqst, "application/octet-stream", file, size, git.token)
	if err != nil {
		return "", err
	}

	return string(body[:]), nil
}

func (md *MetaData) CreateRelease(token string, desc string, filepaths []string) (err errors.Error) {
	release := &gitRelease{
		TagName:    md.SemVer.String(),
		Name:       md.SemVer.String(),
		Prerelease: len(md.SemVer.Prerelease()) != 0,
		Draft:      false,
		Branch:     md.Git.Branch,
		Body:       desc,
		token:      token,
	}

	if len(token) != 0 {
		md.Git.Token = token
	}
	if len(md.Git.Token) == 0 {
		return errors.New("a GITHUB_TOKEN must be present to release to a github repository").With("stack", stack.Trace().TrimRuntime())
	}

	return md.publish(release, filepaths)
}

func (md *MetaData) publish(release *gitRelease, filepaths []string) (err errors.Error) {

	// The github url will have a path where the first item is the user and then the repository name
	parts := strings.Split(md.Git.URL.EscapedPath(), "/")
	if len(parts) != 3 {
		return errors.New("the repository URL has an unexpected number of parts").With("url", md.Git.URL.EscapedPath()).With("stack", stack.Trace().TrimRuntime())
	}
	user := parts[1]
	name := strings.TrimSuffix(parts[len(parts)-1], ".git")

	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", user, name)
	releaseData, errGo := json.Marshal(release)
	if errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	releaseBuffer := bytes.NewBuffer(releaseData)

	data, err := doGitRequest("POST", endpoint, "application/json", releaseBuffer, int64(releaseBuffer.Len()), md.Git.Token)

	if err != nil && data != nil {
		// The release may already exist to rerun the upload assuming it does
		endpoint = fmt.Sprintf("%s/tags/%s", endpoint, release.TagName)
		data, err = doGitRequest("GET", endpoint, "application/json", nil, int64(0), md.Git.Token)
	}

	if err != nil {
		return errors.Wrap(err).With("response", string(data)).With("endpoint", endpoint).With("stack", stack.Trace().TrimRuntime())
	}

	// Gets the release Upload URL from the returned JSON data
	if errGo = json.Unmarshal(data, &release); errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	// Upload URL comes like this https://uploads.github.com/repos/octocat/Hello-World/releases/1/assets{?name}
	// So we need to remove the {?name} part
	uploadURL := strings.Split(release.UploadURL, "{")[0]

	wg := sync.WaitGroup{}

	// Needs refactoring away from wait groups and blind spining off of uploads
	for _, filename := range filepaths {
		wg.Add(1)
		func(file string) {
			// TODO Capture errors and failures for the caller, this is not safe
			// currently
			if resp, err := release.githubUpload(uploadURL, file); err != nil {
				fmt.Println(errors.Wrap(err).With("response", resp).With("stack", stack.Trace().TrimRuntime()).Error())
			}
			wg.Done()
		}(filename)
	}
	wg.Wait()

	return nil
}

// Sends HTTP request to Github API
func doGitRequest(method, url, contentType string, reqBody io.Reader, bodySize int64, token string) (resp []byte, err errors.Error) {

	resp = []byte{}

	req, errGo := http.NewRequest(method, url, reqBody)
	if errGo != nil {
		return resp, errors.Wrap(errGo).With("url", url).With("stack", stack.Trace().TrimRuntime())
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Content-type", contentType)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.ContentLength = bodySize

	httpResp, errGo := http.DefaultClient.Do(req)
	if errGo != nil {
		return resp, errors.Wrap(errGo).With("url", url).With("stack", stack.Trace().TrimRuntime())
	}

	defer httpResp.Body.Close()

	respBody, errGo := ioutil.ReadAll(httpResp.Body)
	if errGo != nil {
		return nil, errors.Wrap(errGo).With("url", url).With("stack", stack.Trace().TrimRuntime())
	}

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		return []byte{}, errors.New("Github error").With("status", httpResp.Status).With("response", respBody).With("url", url).With("stack", stack.Trace().TrimRuntime())
	}

	return respBody, nil
}
