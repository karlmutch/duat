package duat

import (
	"bytes"
	"os/user"
	"strings"
	"testing"

	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"
)

// This file contains a number of test functions for the templating features of duat

// TestUserTemplate exercises the user information functions in the templating
//
func TestUserTemplate(t *testing.T) {

	md, err := NewMetaData(".", "README.md")

	if err != nil {
		t.Fatal(err)
	}

	reader := strings.NewReader("{{.duat.userID}}{{.duat.userName}}{{.duat.userGroupID}}")
	writer := new(bytes.Buffer)

	opts := TemplateOptions{
		IOFiles: []TemplateIOFiles{{
			In:  reader,
			Out: writer,
		}},
		OverrideValues: map[string]string{},
	}

	if err = md.Template(opts); err != nil {
		t.Fatal(err)
	}

	usr, errGo := user.Current()
	if errGo != nil {
		t.Fatal(kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()))
	}
	expected := usr.Uid + usr.Username + usr.Gid
	if expected != writer.String() {
		t.Fatal(kv.NewError("templated user details incorrect").With("expected", expected, "actual", writer.String()).With("stack", stack.Trace().TrimRuntime()))
	}
}
