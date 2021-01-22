package duat

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/eknkc/basex" // MIT License
	"github.com/karlmutch/duat/version"

	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks

	"github.com/karlmutch/semver" // Forked copy of https://github.com/Masterminds/semver

	"github.com/go-stack/stack" // Forked copy of https://github.com/go-stack/stack
	"github.com/jjeffery/kv"    // Forked copy of https://github.com/jjeffery/kv
)

// docHandler allows us to have different document types with regular expressions for each
// that can be used by the code for scraping and saving versions into human readable documents
type docHandler struct {
	ext       string
	fnMatcher *regexp.Regexp
	find      *regexp.Regexp
	replace   *regexp.Regexp
	html      *regexp.Regexp
	subst     string
}

var (
	handlers = []*docHandler{}
)

func (docHandler) GetExts() (exts []string) {
	for _, handler := range handlers {
		exts = append(exts, handler.ext)
	}
	return exts
}

func addHandler(fnMatcher string, find string, replace string, html string, subst string) (err kv.Error) {
	handler := &docHandler{
		ext:   filepath.Ext(fnMatcher),
		subst: subst,
	}

	r, errGo := regexp.Compile(fnMatcher)
	if errGo != nil {
		return kv.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash)
		return
	}
	handler.fnMatcher = r

	if r, errGo = regexp.Compile(find); errGo != nil {
		return kv.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash)
	}
	handler.find = r

	if r, errGo = regexp.Compile(replace); errGo != nil {
		return kv.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash)
	}
	handler.replace = r

	if len(html) == 0 {
		return nil
	}

	if r, errGo = regexp.Compile(html); errGo != nil {
		return kv.Wrap(errGo, "internal error please notify karlmutch@gmail.com").With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash)
	}
	handler.html = r

	handlers = append(handlers, handler)

	return nil
}

func getHandler(fn string) (handler *docHandler, err kv.Error) {
	for _, h := range handlers {
		if h.fnMatcher.Match([]byte(fn)) {
			return h, nil
		}
	}
	return nil, kv.NewError("no version handler for file type").With("file", filepath.Base(fn)).With("stack", stack.Trace().TrimRuntime()).With("version", version.GitHash)
}

func init() {
	if err := addHandler(".*\\.adoc", ":Revision:.*", ":Revision:\\s*(.*)", ":Revision:\\s*", ":Revision: %s"); err != nil {
		fmt.Fprintf(os.Stderr, "asciidoc %v\n", err)
	}

	if err := addHandler(".*\\.md", "\\<repo-version\\>.*?\\</repo-version\\>", "\\<repo-version\\>(.*?)\\</repo-version\\>", "<[^>]*>", "<repo-version>%s</repo-version>"); err != nil {
		fmt.Fprintf(os.Stderr, "markdown %v\n", err)
	}
}

func (md *MetaData) LoadVer(fn string) (ver *semver.Version, err kv.Error) {

	if md.SemVer != nil {
		return nil, kv.NewError("version already loaded").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	handler, err := getHandler(fn)
	if err != nil {
		return nil, err
	}

	file, errGo := os.Open(fn)
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)

	for scan.Scan() {
		versions := handler.find.FindAllString(scan.Text(), -1)
		if len(versions) == 0 {
			continue
		}
		for _, version := range versions {
			if ver == nil {
				extracted := html.UnescapeString(handler.html.ReplaceAllString(version, ""))
				if len(extracted) == 0 {
					continue
				}
				ver, errGo = semver.NewVersion(extracted)
				if errGo != nil {
					return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn).With("extracted", extracted).With("version", version)
				}
				continue
			}
			newVer := html.UnescapeString(handler.html.ReplaceAllString(version, ""))
			if newVer != ver.String() {
				return nil, kv.NewError("all repo-version trimming tags must have the same version string").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
			}
		}
	}

	if ver == nil {
		return nil, kv.NewError("version not found").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	md.SemVer, errGo = semver.NewVersion(ver.String())
	if errGo != nil {
		md.SemVer = nil
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	return ver, nil
}

func (md *MetaData) Apply(files []string) (err kv.Error) {

	if len(files) == 0 {
		return kv.NewError("the apply command requires that files are specified with the -t option").With("stack", stack.Trace().TrimRuntime())
	}

	checkedFiles := make([]string, 0, len(files))
	for _, file := range files {
		if len(file) != 0 {
			if _, errGo := os.Stat(file); errGo != nil {
				return kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", file)
			}
			checkedFiles = append(checkedFiles, file)
		}
	}

	if len(checkedFiles) != len(files) {
		return kv.NewError("no usable targets were found to apply the version to").With("stack", stack.Trace().TrimRuntime())
	}

	// Process the files but stop on any errors
	for _, file := range checkedFiles {
		if err = md.Replace(file, file, false); err != nil {
			return err
		}
	}

	return nil
}

// BumpPrerelease will first bump the release, adn then write the results into
// the file nominated as the version file
//
func (md *MetaData) BumpPrerelease() (result *semver.Version, err kv.Error) {
	if _, err := md.Prerelease(); err != nil {
		return nil, err
	}

	if err := md.Replace(md.VerFile, md.VerFile, false); err != nil {
		return nil, err
	}
	return md.SemVer, nil
}

var (
	alphaEncoder = &basex.Encoding{}
)

func init() {
	alphaEncoder, _ = basex.NewEncoding("abcdefghijkmnopqrstuvwxyz")
}

func (md *MetaData) Prerelease() (result *semver.Version, err kv.Error) {

	if md.Git == nil || md.Git.Err != nil {
		if md.Git.Err != nil {
			return nil, md.Git.Err
		} else {
			return nil, kv.NewError("an operation that required git could not locate git information").With("stack", stack.Trace().TrimRuntime())
		}
	}

	// Generate a pre-release suffix for semver that uses a mixture of the branch name
	// with nothing but hyphens and alpha numerics, followed by a timestamp encoded using
	// semver compatible Base24 in a way that preserves sort ordering and that uses all
	// lower case letters only to respect DNS naming standards
	//
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(time.Now().Unix()))
	build := alphaEncoder.Encode(b)

	// Git branch names can contain characters that would confuse semver including the
	// _ (underscore), and + (plus) characters, https://www.kernel.org/pub/software/scm/git/docs/git-check-ref-format.html
	cleanBranch := ""
	for _, aChar := range md.Git.Branch {
		if aChar < '0' || aChar > 'z' || (aChar > '9' && aChar < 'A') || (aChar > 'Z' && aChar < 'a') {
			cleanBranch += "-"
		} else {
			cleanBranch += string(aChar)
		}
	}
	result = md.SemVer
	newVer, errGo := result.SetPrerelease(fmt.Sprintf("%s-%s", cleanBranch, build))
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	md.SemVer = &newVer

	return md.SemVer, nil
}

func (md *MetaData) Replace(fn string, dest string, substitute bool) (err kv.Error) {

	// To prevent destructive replacements first copy the file then modify the copy
	// and in an atomic operation copy the copy back over the original file, then
	// delete the working file
	origFn, errGo := filepath.Abs(fn)
	if errGo != nil {
		return kv.Wrap(errGo, "input file could not be resolved to an absolute file path").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	tmp, errGo := ioutil.TempFile(filepath.Dir(origFn), filepath.Base(origFn))
	if errGo != nil {
		return kv.Wrap(errGo, "temporary file could not be generated").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	defer func() {
		defer os.Remove(tmp.Name())

		tmp.Close()
	}()

	handler, err := getHandler(fn)
	if err != nil {
		return err
	}

	file, errGo := os.OpenFile(origFn, os.O_RDWR, 0600)
	if errGo != nil {
		return kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	newVer := fmt.Sprintf(handler.subst, md.SemVer.String())
	if substitute {
		newVer = md.SemVer.String()
	}

	scan := bufio.NewScanner(file)
	for scan.Scan() {
		tmp.WriteString(handler.replace.ReplaceAllString(scan.Text(), newVer) + "\n")
	}

	tmp.Sync()
	if fn == dest {
		defer file.Close()
	} else {
		file.Close()

		// Overwrite the output file if it is present
		file, errGo = os.OpenFile(dest, os.O_CREATE|os.O_RDWR, 0600)
		if errGo != nil {
			return kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
		}
		defer file.Close()
	}

	// Ignore kv.if the rewind fails as this could be a stdout style file
	_, _ = file.Seek(0, io.SeekStart)

	if _, errGo = tmp.Seek(0, io.SeekStart); errGo != nil {
		return kv.Wrap(errGo, "failed to rewind a temporary file").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	// Copy the output file on top of the original file
	written, errGo := io.Copy(file, tmp)
	if errGo != nil {
		return kv.Wrap(errGo, "failed to update the output file").With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	// Because we overwrote the file we need to trim off the end of the file if it shrank in size
	file.Truncate(written)

	return nil
}
