package fileio

// Contains the simplified file copy implementation that also supports links

import (
	"io"
	"os"
	"path/filepath"

	"github.com/go-stack/stack" // Forked copy of https://github.com/go-stack/stack
	"github.com/jjeffery/kv"    // Forked copy of https://github.com/jjeffery/kv
)

// CopyFile is used to create a mirror of the source.  If a hard
// link can be used to mirror the file into a new location it will,
// otherwise a new duplicate of the file will be created.
//
func CopyFile(src, dst string) (err kv.Error) {

	sfi, errGo := os.Stat(src)
	if errGo != nil {
		return kv.Wrap(errGo).With("file", src).With("stack", stack.Trace().TrimRuntime())
	}

	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return kv.NewError("cannot copy a non-regular file").With("file", src).With("stack", stack.Trace().TrimRuntime())
	}
	dfi, errGo := os.Stat(dst)
	if errGo != nil {
		if !os.IsNotExist(errGo) {
			return kv.Wrap(errGo, "bad destination file").With("file", dst).With("stack", stack.Trace().TrimRuntime())
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return kv.NewError("cannot copy a non-regular file").With("file", src).With("stack", stack.Trace().TrimRuntime())
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
func copyFileContents(src, dst string) (err kv.Error) {
	in, errGo := os.Open(src)
	if errGo != nil {
		return kv.Wrap(errGo, "missing file").With("file", src).With("stack", stack.Trace().TrimRuntime())
	}
	defer in.Close()
	out, errGo := os.Create(dst)
	if errGo != nil {
		return kv.Wrap(errGo, "destination file not writable").With("file", dst).With("stack", stack.Trace().TrimRuntime())
	}

	defer out.Close()

	if _, errGo = io.Copy(out, in); errGo != nil {
		return kv.Wrap(errGo).With("src", src).With("dst", dst).With("stack", stack.Trace().TrimRuntime())
	}

	if errGo = out.Sync(); errGo != nil {
		return kv.Wrap(errGo, "destination file not sync'ed").With("file", dst).With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}

// IsDir returns true if the given path is an existing directory.
func IsDir(dirPath string) (dirExists bool) {
	if absPath, errGo := filepath.Abs(dirPath); errGo == nil {
		if fileInfo, errGo := os.Stat(absPath); !os.IsNotExist(errGo) && fileInfo.IsDir() {
			return true
		}
	}

	return false
}