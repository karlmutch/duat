package kv

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jjeffery/kv/internal/logfmt"
	"github.com/jjeffery/kv/internal/parse"
	"github.com/jjeffery/kv/internal/pool"
)

// List is a slice of alternating keys and values.
type List []interface{}

// Parse parses the input and reports the message text,
// and the list of key/value pairs.
//
// The text slice, if non-nil, points to the same backing
// array as input.
func Parse(input []byte) (text []byte, list List) {
	m := parse.Bytes(input)
	text = m.Text
	if len(m.List) > 0 {
		list = make(List, len(m.List))
		for i, v := range m.List {
			list[i] = string(v)
		}
	}
	m.Release()
	return text, list
}

// With returns a list populated with keyvals as the key/value pairs.
func With(keyvals ...interface{}) List {
	keyvals = flattenFix(keyvals)
	return List(keyvals)
}

// From returns a new context with key/value pairs copied both from
// the list and the context.
func (l List) From(ctx context.Context) Context {
	ctx = newContext(ctx, l)
	return &contextT{ctx: ctx}
}

// Keyvals returns the list cast as []interface{}.
func (l List) Keyvals() []interface{} {
	return []interface{}(l)
}

// MarshalText implements the TextMarshaler interface.
func (l List) MarshalText() (text []byte, err error) {
	var buf bytes.Buffer
	l.writeToBuffer(&buf)
	return buf.Bytes(), nil
}

// NewError returns an error with the given message and a list of
// key/value pairs copied from the list.
func (l List) NewError(text string) Error {
	e := newError(nil, nil, text)
	e.list = l
	return e
}

// String returns a string representation of the key/value pairs in
// logfmt format: "key1=value1 key2=value2  ...".
func (l List) String() string {
	buf := pool.AllocBuffer()
	defer pool.ReleaseBuffer(buf)
	l.writeToBuffer(buf)
	return buf.String()
}

// UnmarshalText implements the TextUnmarshaler interface.
func (l *List) UnmarshalText(text []byte) error {
	m := parse.Bytes(text)
	defer m.Release()
	capacity := len(m.List)
	if len(m.Text) == 0 {
		capacity += 2
	}
	list := make(List, 0, capacity)

	if len(m.Text) > 0 {
		list = append(list, "msg", string(m.Text))
	}
	for _, v := range m.List {
		list = append(list, string(v))
	}
	*l = list
	return nil
}

// With returns a new list with keyvals appended. The original
// list (l) is not modified.
func (l List) With(keyvals ...interface{}) List {
	keyvals = flattenFix(keyvals)
	list := l.clone(len(l) + len(keyvals))
	list = append(list, keyvals...)
	return list
}

// Wrap wraps the error with the key/value pairs copied from the list,
// and the optional text.
func (l List) Wrap(err error, text ...string) Error {
	e := newError(nil, err, text...)
	e.list = l
	return causer(e)
}

// Log is used to log a message. By default the message is logged
// using the standard logger in the Go "log" package.
func (l List) Log(args ...interface{}) {
	logHelper(2, l, args...)
}

func (l List) clone(capacity int) List {
	length := len(l)
	if capacity < length {
		capacity = length
	}
	list := make(List, length, capacity)
	copy(list, l)
	return list
}

func (l List) writeToBuffer(buf logfmt.Writer) {
	fl := flattenFix(l)
	for i := 0; i < len(fl); i += 2 {
		if i > 0 {
			buf.WriteRune(' ')
		}
		k := fl[i]
		v := fl[i+1]
		logfmt.WriteKeyValue(buf, k, v)
	}
}

func dedup(lists ...List) List {
	var (
		totalLen int
	)
	for _, list := range lists {
		totalLen += len(list)
	}
	if totalLen == 0 {
		return nil
	}
	result := make(List, 0, totalLen)
	m := make(map[string]map[string]struct{})
	buf := pool.AllocBuffer()
	defer pool.ReleaseBuffer(buf)

	valueString := func(val interface{}) string {
		buf.Reset()
		logfmt.WriteValue(buf, val)
		return buf.String()
	}

	for _, list := range lists {
		contents := flattenFix(list)
		for i := 0; i < len(contents); i += 2 {
			key, ok := contents[i].(string)
			if !ok {
				// shouldn't happen, unless a different type is
				// returned for missing keys, which might happen
				// if the flatten/fix function is modified in future
				key = fmt.Sprint(key)
			}
			val := contents[i+1]
			valstr := valueString(val)
			valstrs, found := m[key]
			if !found {
				valstrs = make(map[string]struct{})
				m[key] = valstrs
			}
			if _, ok := valstrs[valstr]; !ok {
				result = append(result, key, val)
				valstrs[valstr] = struct{}{}
			}
		}
	}

	return result
}
