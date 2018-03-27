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

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

// createTest is used to generate a temporary file that the caller should delete after
// running the test/  The temporary file will contain a version string, if specified, and
// appropriate tags for the version along with other text.
//
func createTestFile(version *semver.Version) (fn string, err errors.Error) {
	content := []byte("content <repo-version></repo-version> content")
	if version != nil {
		content = []byte(fmt.Sprintf("content <repo-version>%s</repo-version> content", version.String()))
	}
	tmpfile, errGo := ioutil.TempFile("", "test-extract")
	if errGo != nil {
		return "", errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()).With("file", tmpfile.Name())
	}

	fn = tmpfile.Name()
	if _, errGo = tmpfile.Write(content); err != nil {
		msg := fmt.Sprintf("test file %s could not be used for storing test data", tmpfile.Name())
		return fn, errors.Wrap(errGo, msg).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
	}
	if errGo = tmpfile.Close(); errGo != nil {
		msg := fmt.Sprintf("test file %s could not be closed", tmpfile.Name())
		return fn, errors.Wrap(errGo, msg).With("stack", stack.Trace().TrimRuntime()).With("file", fn)
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

	if err := ApplyCase(ver, major); err != nil {
		t.Error(err)
		return
	}
	if err := ApplyCase(nil, major); err != nil {
		t.Error(err)
		return
	}
}

func ApplyCase(fileVer *semver.Version, applyVer semver.Version) (err errors.Error) {
	fn, err := createTestFile(fileVer)
	if err != nil {
		return errors.Wrap(err, "unable to create test file for apply test").With("stack", stack.Trace().TrimRuntime()).With("ver", applyVer)
	}
	defer os.Remove(fn)

	md := &MetaData{
		SemVer: &applyVer,
	}
	err = md.Apply([]string{fn})
	if err != nil {
		return errors.Wrap(err, "unable to create test file for apply test").With("stack", stack.Trace().TrimRuntime()).With("ver", applyVer)
	}

	newMD := &MetaData{}
	result, err := newMD.LoadVer(fn)
	if err != nil {
		return errors.Wrap(err, "failed to apply major version change to version").With("stack", stack.Trace().TrimRuntime()).With("ver", applyVer)
	}
	if result.String() != newMD.SemVer.String() {
		return errors.Wrap(err, "applying the major version change failed to return the same value as that stored").With("stack", stack.Trace().TrimRuntime()).With("ver", applyVer)
	}

	if applyVer.String() != newMD.SemVer.String() {
		return errors.Wrap(err, "applying the major version change failed to return the expected value").With("stack", stack.Trace().TrimRuntime()).With("ver", applyVer)
	}
	return nil
}

func TestVersionReplace(t *testing.T) {

	ver, err := semver.NewVersion("0.0.0-pre+build")
	if err != nil {
		t.Error(fmt.Errorf("unable to parse test data due to %v", err))
	}

	srcFn, err := createTestFile(ver)
	if err != nil {
		t.Error(fmt.Errorf("unable to create test file for replace test due to %v", err))
		return
	}

	defer func() {
		if !t.Failed() {
			os.Remove(srcFn)
		}
	}()

	patch := ver.IncPatch()
	md := &MetaData{
		SemVer: &patch,
	}

	emptyDestFn, err := createTestFile(nil)
	if err != nil {
		t.Error(fmt.Errorf("unable to create test file for replace test due to %v", err))
		return
	}

	defer func() {
		if !t.Failed() {
			os.Remove(emptyDestFn)
		}
	}()

	// Replace versions
	if err = md.Replace(srcFn, emptyDestFn, false); err != nil {
		t.Error(fmt.Errorf("unable to replace test file %s into %s due to %v", srcFn, emptyDestFn, err))
		return
	}

	// From the src we should be able to get the original version
	originalMD := &MetaData{}
	if _, err = originalMD.LoadVer(srcFn); err != nil {
		t.Error(errors.Wrap(err, "failed to read the version from the original file").With("stack", stack.Trace().TrimRuntime()))
		return
	}
	if ver.String() != originalMD.SemVer.String() {
		t.Error(errors.Wrap(err, "the original file was modified unexpectly").With("stack", stack.Trace().TrimRuntime()))
		return
	}

	// From the dest we must get the new version
	replacedMD := &MetaData{}
	if _, err = replacedMD.LoadVer(emptyDestFn); err != nil {
		t.Error(errors.Wrap(err, "failed to read the version from the processed file").With("stack", stack.Trace().TrimRuntime()).With("srcFn", srcFn))
		return
	}
	if md.SemVer.String() != replacedMD.SemVer.String() {
		t.Error(errors.Wrap(err, "the processed file was not correctly modified").With("stack", stack.Trace().TrimRuntime()))
		return
	}

	// Now use a destination that already has a version and make sure it gets replaced
	minor := ver.IncMinor()
	mdMinor := &MetaData{
		SemVer: &minor,
	}

	minorDestFn, err := createTestFile(&minor)
	if err != nil {
		t.Error(fmt.Errorf("unable to create test file for replace test due to %v", err))
		return
	}

	defer func() {
		if !t.Failed() {
			os.Remove(minorDestFn)
		}
	}()

	minorReplacedMD := &MetaData{}
	if _, err = minorReplacedMD.LoadVer(minorDestFn); err != nil {
		t.Error(errors.Wrap(err, "failed to read the minor version from the minor version change populated file").With("stack", stack.Trace().TrimRuntime()).With("srcFn", srcFn))
		return
	}
	if mdMinor.SemVer.String() != minorReplacedMD.SemVer.String() {
		t.Error(errors.Wrap(err, "the already populated file was not correctly modified").With("stack", stack.Trace().TrimRuntime()))
		return
	}
}
