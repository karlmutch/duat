package duat

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	// The following packages are forked to retain copies in the event github accounts are shutdown
	//
	// I am torn between this and just letting dep ensure with a checkedin vendor directory
	// to do this.  In any event I ended up doing both with my own forks

	"github.com/karlmutch/semver" // Forked copy of https://github.com/Masterminds/semver

	"github.com/go-stack/stack" // Forked copy of https://github.com/go-stack/stack
	"github.com/jjeffery/kv"    // Forked copy of https://github.com/jjeffery/kv
)

// createTest is used to generate a temporary file that the caller should delete after
// running the test/  The temporary file will contain a version string, if specified, and
// appropriate tags for the version along with other text.
//
func createTestFile(version *semver.Version, fileExt string) (fn string, err kv.Error) {
	content := []byte{}

	switch fileExt {
	case ".adoc":
		content = []byte("content\n:Revision:\ncontent")
		if version != nil {
			content = []byte(fmt.Sprintf("content\n:Revision: %s\ncontent", version.String()))
		}
	case ".md":
		content = []byte("content <repo-version></repo-version> content")
		if version != nil {
			content = []byte(fmt.Sprintf("content <repo-version>%s</repo-version> content", version.String()))
		}
	default:
		fmt.Println("Unknown extension", fileExt, "stack", stack.Trace().TrimRuntime())
	}
	tmpfile, errGo := ioutil.TempFile("", "test-extract-*"+fileExt)
	if errGo != nil {
		return "", kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", tmpfile.Name())
	}

	fn = tmpfile.Name()
	if _, errGo = tmpfile.Write(content); err != nil {
		msg := fmt.Sprintf("test file %s could not be used for storing test data", tmpfile.Name())
		return fn, kv.Wrap(errGo, msg).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	if errGo = tmpfile.Close(); errGo != nil {
		msg := fmt.Sprintf("test file %s could not be closed", tmpfile.Name())
		return fn, kv.Wrap(errGo, msg).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}

	return fn, nil
}

func TestVersionApply(t *testing.T) {

	// Apply is used to modify a file in place to include a version.

	// First use a version and write the file with the old version.  Then generate a new
	// version and apply it to the file.  Final,ly validating the content of the file
	// to ensure is was modified.
	//
	ver, err := semver.NewVersion("0.0.0-pre+build")
	if err != nil {
		t.Error(fmt.Errorf("unable to parse test data due to %v", err))
		return
	}
	major := ver.IncMajor()

	for _, ext := range (docHandler{}).GetExts() {
		if err := ApplyCase(ver, major, ext); err != nil {
			t.Error(err)
			return
		}
		if err := ApplyCase(nil, major, ext); err != nil {
			t.Error(err)
			return
		}
	}
}

func ApplyCase(fileVer *semver.Version, applyVer semver.Version, fileExt string) (err kv.Error) {

	failed := true

	fn, err := createTestFile(fileVer, fileExt)
	if err != nil {
		return kv.Wrap(err, "unable to create test file for apply test").With("stack", stack.Trace().TrimRuntime()).With("ver", applyVer)
	}
	defer func() {
		if failed {
			return
		}
		os.Remove(fn)
	}()

	md := &MetaData{
		SemVer: &applyVer,
	}
	err = md.Apply([]string{fn})
	if err != nil {
		return kv.Wrap(err, "unable to create test file for apply test").With("file", fn).With("stack", stack.Trace().TrimRuntime()).With("ver", applyVer)
	}

	newMD := &MetaData{}
	result, err := newMD.LoadVer(fn)
	if err != nil {
		return kv.Wrap(err, "failed to apply major version change").With("file", fn).With("stack", stack.Trace().TrimRuntime()).With("ver", applyVer)
	}
	if result.String() != newMD.SemVer.String() {
		return kv.Wrap(err, "applying the major version change failed to return the same value as that stored").With("file", fn).With("stack", stack.Trace().TrimRuntime()).With("ver", applyVer)
	}

	if applyVer.String() != newMD.SemVer.String() {
		return kv.Wrap(err, "applying the major version change failed to return the expected value").With("file", fn).With("stack", stack.Trace().TrimRuntime()).With("ver", applyVer)
	}

	failed = false
	return nil
}

func TestVersionReplace(t *testing.T) {

	ver, err := semver.NewVersion("0.0.0-pre+build")
	if err != nil {
		t.Error(fmt.Errorf("unable to parse test data due to %v", err))
	}

	for _, ext := range (docHandler{}).GetExts() {
		srcFn, err := createTestFile(ver, ext)
		if err != nil {
			t.Error(fmt.Errorf("unable to create test file for replace test due to %v", err))
			return
		}

		defer func() {
			if t.Failed() {
				return
			}
			os.Remove(srcFn)
		}()

		patch := ver.IncPatch()
		md := &MetaData{
			SemVer: &patch,
		}

		fmt.Println("Doing empty version string test", ext)
		emptyDestFn, err := createTestFile(nil, ext)
		if err != nil {
			t.Error(fmt.Errorf("unable to create test file for replace test due to %v", err))
			return
		}

		defer func() {
			if t.Failed() {
				return
			}
			os.Remove(emptyDestFn)
		}()

		// Replace versions
		if err = md.Replace(srcFn, emptyDestFn, false); err != nil {
			t.Error(fmt.Errorf("unable to replace test file %s into %s due to %v", srcFn, emptyDestFn, err))
			return
		}

		// From the src we should be able to get the original version
		originalMD := &MetaData{}
		if _, err = originalMD.LoadVer(srcFn); err != nil {
			t.Error(kv.Wrap(err, "failed to read the version from the original file").With("fn", srcFn).With("stack", stack.Trace().TrimRuntime()))
			return
		}
		if ver.String() != originalMD.SemVer.String() {
			t.Error(kv.Wrap(err, "the original file was modified unexpectly").With("fn", srcFn).With("stack", stack.Trace().TrimRuntime()))
			return
		}

		// From the dest we must get the new version
		replacedMD := &MetaData{}
		if _, err = replacedMD.LoadVer(emptyDestFn); err != nil {
			t.Error(kv.Wrap(err, "failed to read the version from the processed file").With("fn", emptyDestFn).With("stack", stack.Trace().TrimRuntime()).With("srcFn", srcFn))
			return
		}
		if md.SemVer.String() != replacedMD.SemVer.String() {
			t.Error(kv.Wrap(err, "the processed file was not correctly modified").With("fn", emptyDestFn).With("stack", stack.Trace().TrimRuntime()))
			return
		}

		// Now use a destination that already has a version and make sure it gets replaced
		minor := ver.IncMinor()
		mdMinor := &MetaData{
			SemVer: &minor,
		}

		minorDestFn, err := createTestFile(&minor, ext)
		if err != nil {
			t.Error(fmt.Errorf("unable to create test file for replace test due to %v", err))
			return
		}

		defer func() {
			if t.Failed() {
				return
			}
			os.Remove(minorDestFn)
		}()

		minorReplacedMD := &MetaData{}
		if _, err = minorReplacedMD.LoadVer(minorDestFn); err != nil {
			t.Error(kv.Wrap(err, "failed to read the minor version from the minor version change populated file").With("fn", minorDestFn).With("stack", stack.Trace().TrimRuntime()).With("srcFn", srcFn))
			return
		}
		if mdMinor.SemVer.String() != minorReplacedMD.SemVer.String() {
			t.Error(kv.Wrap(err, "the already populated file was not correctly modified").With("fn", minorDestFn).With("stack", stack.Trace().TrimRuntime()))
			return
		}
	}
}
