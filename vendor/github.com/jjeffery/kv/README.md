# kv [![GoDoc](https://godoc.org/github.com/jjeffery/kv?status.svg)](https://godoc.org/github.com/jjeffery/kv) [![License](http://img.shields.io/badge/license-MIT-green.svg?style=flat)](https://raw.githubusercontent.com/jjeffery/kv/master/LICENSE.md) [![Build Status](https://travis-ci.org/jjeffery/kv.svg?branch=master)](https://travis-ci.org/jjeffery/kv) [![Coverage Status](https://coveralls.io/repos/github/jjeffery/kv/badge.svg?branch=master)](https://coveralls.io/github/jjeffery/kv?branch=master) [![GoReportCard](https://goreportcard.com/badge/github.com/jjeffery/kv)](https://goreportcard.com/report/github.com/jjeffery/kv)

Package kv provides support for working with collections of key/value pairs.

- [Lists](#lists)
- [Errors](#errors)
- [Context](#context)
- [Parse](#parse)
- [Logging](#logging)

## Lists

The `List` type represents a sequence of alternating key/value pairs. Lists
can render themselves as text in [logfmt](https://brandur.org/logfmt) format, 
so if you like logging messages with key value pairs but are not ready to move 
away from the standard library log package you can use something like:
```go
log.Println("this is a log message", kv.With(
    "key1", "value 1",
    "key2", 2,
))

// Output:
// this is a log message key1="value 1" key2=2
```

The output from the previous example can be easily read by humans, and easily [parsed](#parse)
by machines. This makes it an excellent format for 
[structured logging](https://www.thoughtworks.com/radar/techniques/structured-logging).

## Errors

The `Error` type implements the builtin `error` interface and renders its error message as a
free text message followed by key/value pairs:
```
cannot open file: permission denied file="/etc/passwd" user=alice
```

Errors are constructed with message text and key/value pairs.
```go
// create an error
err := kv.NewError("permission denied").With("user", user)
log.Println(err)

// Output:
// permission denied user=alice
```

Errors can wrap other errors:
```go
err = kv.Wrap(err, "cannot open file").With("file", filename)
log.Println(err)

// Output:
// cannot open file: permission denied file="/etc/passwd" user=alice
```

## Context

Key/value pairs can be stored in the context:
```go
ctx := context.Background()

// associate some key/value pairs with the context
ctx = kv.From(ctx).With("url", "/api/widgets", "method", "get")

// ... later on ...

log.Println("access denied", kv.From(ctx).With("file", filename))

// Output:
// access denied file="/etc/passwd" url="/api/widgets" method=get
```

## Parse

One of the key points of structured logging is that logs are machine
readable. The `Parse` function provides an easy way to read messages.
```go
// read one line
line := []byte(`cannot open file: access denied file="/etc/passwd" user=alice`)

 text, list := kv.Parse(line)
 log.Println("text:", string(text))
 log.Println("list:", list)

// Output:
// text: cannot open file: access denied
// list: file="/etc/passwd" user=alice
```

## Logging

The `kvlog` subdirectory contains a package that works well with the `log` package
in the Go standard library. Its usage is as simple as:
```go
func main() {
    kvlog.Attach() // attach to the standard logger
    log.Println("program started", kv.With("args", os.Args))
}
```

See the [GoDoc](https://godoc.org/github.com/jjeffery/kv) for more details.
