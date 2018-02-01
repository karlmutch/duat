/*
Package kv provides support for working with collections of key/value pairs.
The package provides types for a pair, list and map of key/value pairs.

Rendering as text

The types Pair, List and Map all implement the fmt.String interface and the
encoding.TextMarshaler interface, and so they can render themselves as text.
For example if you like the simplicity of logging with key value pairs but
are not ready to move away from the standard library log package you can
use this package to render your key value pairs.
  log.Println("this is a log message", kv.List{
      "key1", "value 1",
      "key2", 2,
  })

  // Output (not including prefixes added by the log package):
  // this is a log message key1="value 1" key2=2

This works well with the colog package (https://godoc.org/comail.io/go/colog).

Flattening and Fixing

The kv.Flatten function accepts a slice of interface{} and "flattens" it to
return a slice with an even-numbered length where the value at every even-numbered
index is a keyword string. It can flatten arrays:

 // ["k1", 1, "k2", 2, "k3", 3, "k4", 4]
 keyvals := kv.Flatten({"k1", 1, []interface{}{"k2", 2, "k3", 3}, "k4", 4})

Flatten is reasonably good at working out what to do when
the input length is not an even number, or when one of the items at an even-numbered
index is not a string value. For example, it will infer a message value without
a key and give it a "msg" key.

 // ["msg", "message 1", "key1", 1, "key2", 2]
 keyvals := kv.Flatten("message 1", kv.Map{
     "key1": 1,
     "key2": 2,
 }))

If a value is present without a key it will assign it an arbitrary one.

 // ["msg", "message 2", "key3", 3, "_p1", 4]
 keyvals := kv.Flatten("msg", "message 3", "key3", 3, 4)

A single error gets turned into a message.

 // ["msg", "the error message"]
 keyvals = kv.Flatten(err)

See the Flatten tests for more examples of how kv.Flatten will attempt to
fix non-conforming key/value lists. (https://github.com/jjeffery/kv/blob/master/flatten_test.go)

The keyvalser interface

The List, Map and Pair types all implement the following interface:

 type keyvalser interface {
     Keyvals() []interface{}
 }

The Flatten function recognises types that implement this interface and treats
them as a slice of key/value pairs when flattening a slice.
*/
package kv
