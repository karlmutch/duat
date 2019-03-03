package kv

import (
	"context"
	"fmt"
	"time"

	"github.com/jjeffery/kv/internal/pool"
)

// ctxKeyT is the type used for the context keys.
type ctxKeyT string

// ctxKey is the key used for storing key/value pairs
// in the context.
var ctxKey ctxKeyT = "kv"

// Context implements the context.Context interface,
// and can create a new context with key/value pairs
// attached to it.
type Context interface {
	context.Context

	// With returns a new context based on the existing context,
	// but with with the key/value pairs attached.
	With(keyvals ...interface{}) context.Context

	// NewError returns a new error with the message text and
	// the key/value pairs from the context attached.
	NewError(text string) Error

	// Wrap returns an error that wraps the existing error with
	// the optional message text, and the key/value pairs from
	// the context attached.
	Wrap(err error, text ...string) Error
}

// contextT implements the Context interface.
type contextT struct {
	ctx context.Context
}

// From creates a new context based on ctx. See the Context example.
func From(ctx context.Context) Context {
	if c, ok := ctx.(Context); ok {
		return c
	}
	return &contextT{ctx: ctx}
}

// Deadline implements the context.Context interface.
func (c *contextT) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

// Done implements the context.Context interface.
func (c *contextT) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err implements the context.Context interface.
func (c *contextT) Err() error {
	return c.ctx.Err()
}

// Value implements the context.Context interface.
func (c *contextT) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}

// With returns a context.Context with the keyvals attached.
func (c *contextT) With(keyvals ...interface{}) context.Context {
	return &contextT{ctx: newContext(c.ctx, keyvals)}
}

func newContext(ctx context.Context, keyvals []interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(keyvals) == 0 {
		return ctx
	}
	keyvals = keyvals[:len(keyvals):len(keyvals)] // set capacity
	keyvals = flattenFix(keyvals)
	keyvals = append(keyvals, fromContext(ctx)...)
	keyvals = keyvals[:len(keyvals):len(keyvals)] // set capacity
	return context.WithValue(ctx, ctxKey, keyvals)
}

func fromContext(ctx context.Context) []interface{} {
	var keyvals []interface{}
	if ctx != nil {
		keyvals, _ = ctx.Value(ctxKey).([]interface{})
	}
	return keyvals
}

func (c *contextT) NewError(text string) Error {
	return newError(c.ctx, nil, text)
}

func (c *contextT) Wrap(err error, text ...string) Error {
	return newError(c.ctx, err, text...)
}

func (c *contextT) String() string {
	list := List(fromContext(c.ctx))
	return list.String()
}

// Format implements the fmt.Formatter interface. If
// the context is printed with "%+v", then it prints
// using the String method of the wrapped context.
func (c *contextT) Format(f fmt.State, ch rune) {
	if ch == 'v' && f.Flag('+') {
		fmt.Fprint(f, c.ctx)
		return
	}
	buf := pool.AllocBuffer()
	list := List(fromContext(c.ctx))
	list.writeToBuffer(buf)
	f.Write(buf.Bytes())
	pool.ReleaseBuffer(buf)
}
