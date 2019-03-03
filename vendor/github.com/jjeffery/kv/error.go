package kv

import (
	"context"
	"strings"

	"github.com/jjeffery/kv/internal/pool"
)

// Error implements the builtin error interface, and provides
// an additional method for attaching key/value pairs.
type Error interface {
	error

	// With returns a new error based on this error
	// with the key/value pairs attached.
	With(keyvals ...interface{}) Error
}

type errorT struct {
	text    string
	list    List
	ctxlist List
	err     error
}

var _ Error = &errorT{}

// NewError returns an error that formats as the given text.
func NewError(text string) Error {
	return newError(nil, nil, text)
}

// Wrap returns an error that wraps err, optionally annotating
// with the message text.
func Wrap(err error, text ...string) Error {
	return causer(newError(nil, err, text...))
}

func newError(ctx context.Context, err error, text ...string) *errorT {
	e := &errorT{
		text: strings.Join(text, " "),
		err:  err,
	}
	e.ctxlist = append(e.ctxlist, fromContext(ctx)...)
	return e
}

// Error implements the error interface.
//
// The string returned prints the error text of this error
// any any wrapped errors, each separated by a colon and a space (": ").
// After the error message (or messages) comes the key/value pairs.
// The resulting string can be parsed with the Parse function.
func (e *errorT) Error() string {
	var (
		text     = strings.TrimSpace(e.text)
		prevText []byte
		prevList List
	)
	if e.err != nil {
		input := []byte(e.err.Error())
		prevText, prevList = Parse(input)
		if len(prevText) == 0 && len(prevList) > 0 {
			// The previous message consists only of key/value
			// pairs. Search for a key indicating the message.
			i := 0
			newLen := len(prevList)
			for ; i < len(prevList); i += 2 {
				key, _ := prevList[i].(string)
				if key == "msg" {
					if value, ok := prevList[i+1].(string); ok {
						prevText = []byte(value)
						newLen -= 2
						break
					}
				}
			}
			for ; i < len(prevList)-2; i += 2 {
				prevList[i] = prevList[i+2]
				prevList[i+1] = prevList[i+3]
			}
			prevList = prevList[:newLen]
		}
	}

	buf := pool.AllocBuffer()
	defer pool.ReleaseBuffer(buf)

	if len(text) > 0 {
		buf.WriteString(text)
		if len(prevText) > 0 {
			buf.WriteString(": ")
		}
	}
	buf.Write(prevText)

	list := dedup(e.list, prevList, e.ctxlist)
	if len(list) > 0 {
		if buf.Len() > 0 {
			buf.WriteRune(' ')
		}
		list.writeToBuffer(buf)
	}

	return buf.String()
}

func (e *errorT) With(keyvals ...interface{}) Error {
	return causer(&errorT{
		text:    e.text,
		list:    e.list.With(keyvals...),
		ctxlist: e.ctxlist,
		err:     e.err,
	})
}

// Unwrap implements the Wrapper interface.
// See golang.org/x/exp/errors.
func (e *errorT) Unwrap() error {
	return e.err
}

// causerT implements the causer interface.
// See github.com/pkg/errors.
type causerT struct {
	*errorT
}

// Cause implements the causer interface.
func (e *causerT) Cause() error {
	return e.errorT.err
}

// causer detects whether e wraps another error, and if so
// returns an Error that also implements the Causer interface.
func causer(e *errorT) Error {
	if e.err == nil {
		return e
	}
	return &causerT{
		errorT: e,
	}
}
