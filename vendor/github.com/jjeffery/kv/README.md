# kv [![GoDoc](https://godoc.org/github.com/jjeffery/kv?status.svg)](https://godoc.org/github.com/jjeffery/kv) [![License](http://img.shields.io/badge/license-MIT-green.svg?style=flat)](https://raw.githubusercontent.com/jjeffery/kv/master/LICENSE.md) [![Build Status](https://travis-ci.org/jjeffery/kv.svg?branch=master)](https://travis-ci.org/jjeffery/kv) [![Coverage Status](https://coveralls.io/repos/github/jjeffery/kv/badge.svg?branch=master)](https://coveralls.io/github/jjeffery/kv?branch=master)

Package kv provides support for working with collections of key/value pairs. 

It provides `List`, `Pair` and `Map` types for keyword/value lists, pairs and maps.

```go
log.Println("info: this is a log message", kv.List{
    "key1", "value 1",
    "key2", 2,
})

// Output:
// 2009/11/10 12:34:56 info: this is a log message key1="value 1" key2=2
```

See the [GoDoc](https://godoc.org/github.com/jjeffery/kv) for more details.
