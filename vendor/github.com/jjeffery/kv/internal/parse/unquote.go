package parse

import (
	"bytes"
	"strconv"
	"unicode/utf8"
	"unsafe"
)

var (
	errorIndicator = []byte("???")
)

// unquote the input. If possible the unquoted value points to the same
// backing array as input. Otherwise it points to buf. The remainder is
// the unused portion of buf.
func unquote(input []byte, buf []byte) (unquoted []byte, remainder []byte) {
	if len(input) < 2 {
		return errorIndicator, buf
	}
	quote := input[0]
	input = input[1:]
	if input[len(input)-1] == quote {
		input = input[:len(input)-1]
	}
	index := bytes.IndexRune(input, '\\')
	if index < 0 {
		// input does not contain any escaped chars
		remainder = buf
		unquoted = input
		return unquoted, remainder
	}
	if len(buf) > 0 {
		unquoted = buf[:0]
	}
	// BUG(jpj): Uses the unsafe package to cast a byte slice to a string.
	// This could break in future versions of Go. The reason for this conversion
	// is that there is no equivalent to strconv.UnquoteChar that works on a byte slice.
	strinput := toString(input)
	for len(strinput) > 0 {
		r, mb, tail, err := strconv.UnquoteChar(strinput, quote)
		if err != nil {
			return errorIndicator, buf
		}
		strinput = tail
		if mb {
			// ensure that there is enough room for the multibyte char
			runeLen := utf8.RuneLen(r)
			unquotedLen := len(unquoted)
			for i := 0; i < runeLen; i++ {
				unquoted = append(unquoted, 0)
			}
			utf8.EncodeRune(unquoted[unquotedLen:], r)
		} else {
			unquoted = append(unquoted, byte(r))
		}
	}

	if len(buf) < len(unquoted) {
		// used buf up and resorted to memory allocation
		remainder = nil
	} else {
		remainder = buf[len(unquoted):]
	}

	return unquoted, remainder
}

func toString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
