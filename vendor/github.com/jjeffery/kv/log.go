package kv

import (
	"context"
	"fmt"
	"log"
)

var (
	// LogOutput is the function called to log a message.
	// The default value calls the Output function for the
	// standard logger in the Go standard liberary.
	LogOutput = log.Output
)

// Log is used to log a message. By default the message is logged
// using the standard logger in the Go "log" package.
func Log(args ...interface{}) {
	logHelper(2, nil, args...)
}

func logHelper(calldepth int, list List, args ...interface{}) {
	var ctx context.Context
	var lists []List
	var others []interface{}

	if list != nil {
		lists = append(lists, list)
	}

	for _, arg := range args {
		switch v := arg.(type) {
		case context.Context:
			ctx = From(v)
		case List:
			lists = append(lists, v)
		default:
			others = append(others, arg)
		}
	}

	if ctx != nil {
		keyvals := fromContext(ctx)
		if len(keyvals) > 0 {
			lists = append(lists, List(keyvals))
		}
	}

	others = append(others, dedup(lists...))
	s := fmt.Sprintln(others...)
	LogOutput(calldepth+1, s)
}
