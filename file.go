package duat

// Contains the simplified file copy implementation that also supports links

import (
	"io"
	"os"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

// CopyFile is used to create a mirror of the source.  If a hard
// link can be used to mirror the file into a new location it will,
// otherwise a new duplicate of the file will be created.
//
func CopyFile(src, dst string) (err errors.Error) {

	sfi, errGo := os.Stat(src)
	if errGo != nil {
		return errors.Wrap(errGo).With("file", src).With("stack", stack.Trace().TrimRuntime())
	}

	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return errors.New("cannot copy a non-regular file").With("file", src).With("stack", stack.Trace().TrimRuntime())
	}
	dfi, errGo := os.Stat(dst)
	if errGo != nil {
		if !os.IsNotExist(errGo) {
			return errors.Wrap(errGo, "bad destination file").With("file", dst).With("stack", stack.Trace().TrimRuntime())
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return errors.New("cannot copy a non-regular file").With("file", src).With("stack", stack.Trace().TrimRuntime())
		}
		if os.SameFile(sfi, dfi) {
			return nil
		}
	}

	if errGo = os.Link(src, dst); errGo == nil {
		return nil
	}
	return copyFileContents(src, dst)
}

// copyFileContents creates a true duplicate file
//
func copyFileContents(src, dst string) (err errors.Error) {
	in, errGo := os.Open(src)
	if errGo != nil {
		return errors.Wrap(errGo, "missing file").With("file", src).With("stack", stack.Trace().TrimRuntime())
	}
	defer in.Close()
	out, errGo := os.Create(dst)
	if errGo != nil {
		return errors.Wrap(errGo, "destination file not writable").With("file", dst).With("stack", stack.Trace().TrimRuntime())
	}

	defer out.Close()

	if _, errGo = io.Copy(out, in); errGo != nil {
		return errors.Wrap(errGo).With("src", src).With("dst", dst).With("stack", stack.Trace().TrimRuntime())
	}

	if errGo = out.Sync(); errGo != nil {
		return errors.Wrap(errGo, "destination file not sync'ed").With("file", dst).With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}
