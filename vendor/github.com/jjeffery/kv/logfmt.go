package kv

import (
	"bytes"
	"encoding"
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

// constant byte values
var (
	bytesNull   = []byte("null")
	bytesPanic  = []byte(`<PANIC>`)
	bytesError  = []byte(`<ERROR>`)
	bytesEmptyK = []byte(`<EMPTY>`)
	bytesEmptyV = []byte(`""`)

	escapeSequences = map[rune]string{
		'\t': `\t`,
		'\r': `\r`,
		'\n': `\n`,
		'"':  `\"`,
		'\\': `\\`,
	}
)

func writeKeyValue(buf *bytes.Buffer, key, value interface{}) {
	if buf.Len() > 0 {
		buf.WriteRune(' ')
	}
	writeKey(buf, key)
	buf.WriteRune('=')
	writeValue(buf, value)
}

func writeKey(buf *bytes.Buffer, value interface{}) {
	defer func() {
		if r := recover(); r != nil {
			if buf != nil {
				buf.Write(bytesPanic)
			}
		}
	}()
	switch v := value.(type) {
	case nil:
		writeBytesKey(buf, bytesNull)
		return
	case []byte:
		writeBytesKey(buf, v)
		return
	case string:
		writeStringKey(buf, v)
		return
	case bool, byte, int8, int16, uint16, int32, uint32, int64, uint64, int, uint, uintptr, float32, float64, complex64, complex128:
		fmt.Fprint(buf, v)
		return
	case encoding.TextMarshaler:
		writeTextMarshalerKey(buf, v)
		return
	case error:
		writeStringKey(buf, v.Error())
		return
	case fmt.Stringer:
		writeStringKey(buf, v.String())
		return
	default:
		// handle pointer to any of the above
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				buf.Write(bytesNull)
				return
			}
			writeKey(buf, rv.Elem().Interface())
			return
		}
		writeStringKey(buf, fmt.Sprint(value))
	}
}

func writeBytesKey(buf *bytes.Buffer, b []byte) {
	if b == nil {
		buf.Write(bytesNull)
		return
	}
	if len(b) == 0 {
		buf.Write(bytesEmptyK)
		return
	}
	for {
		index := bytes.IndexFunc(b, invalidKey)
		if index < 0 {
			break
		}
		if index > 0 {
			buf.Write(b[:index])
			b = b[index:]
		}
		buf.WriteRune('_')
		// we know that the rune will be a single byte
		b = b[1:]
	}
	buf.Write(b)
}

func writeStringKey(buf *bytes.Buffer, s string) {
	if s == "" {
		buf.Write(bytesEmptyK)
		return
	}
	index := strings.IndexFunc(s, invalidKey)
	if index < 0 {
		buf.WriteString(s)
		return
	}
	for _, c := range s {
		if invalidKey(c) {
			buf.WriteRune('_')
		} else {
			buf.WriteRune(c)
		}
	}
}

func writeTextMarshalerKey(buf *bytes.Buffer, t encoding.TextMarshaler) {
	b, err := t.MarshalText()
	if err != nil {
		buf.Write(bytesError)
		return
	}
	writeBytesKey(buf, b)
}

func writeValue(buf *bytes.Buffer, value interface{}) {
	defer func() {
		if r := recover(); r != nil {
			if buf != nil {
				buf.Write(bytesPanic)
			}
		}
	}()
	switch v := value.(type) {
	case nil:
		writeBytesValue(buf, bytesNull)
		return
	case []byte:
		writeBytesValue(buf, v)
		return
	case string:
		writeStringValue(buf, v)
		return
	case bool, byte, int8, int16, uint16, int32, uint32, int64, uint64, int, uint, uintptr, float32, float64, complex64, complex128:
		fmt.Fprint(buf, v)
		return
	case encoding.TextMarshaler:
		writeTextMarshalerValue(buf, v)
		return
	case error:
		writeStringValue(buf, v.Error())
		return
	case fmt.Stringer:
		writeStringValue(buf, v.String())
		return
	default:
		// handle pointer to any of the above
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				buf.Write(bytesNull)
				return
			}
			writeValue(buf, rv.Elem().Interface())
			return
		}
		writeStringValue(buf, fmt.Sprint(value))
	}
}

func writeBytesValue(buf *bytes.Buffer, b []byte) {
	if b == nil {
		buf.Write(bytesNull)
		return
	}
	if len(b) == 0 {
		buf.Write(bytesEmptyV)
		return
	}
	index := bytes.IndexFunc(b, needsQuote)
	if index < 0 {
		buf.Write(b)
		return
	}
	buf.WriteRune('"')
	if index > 0 {
		buf.Write(b[0:index])
		b = b[index:]
	}
	for {
		index = bytes.IndexFunc(b, needsBackslash)
		if index < 0 {
			break
		}
		if index > 0 {
			buf.Write(b[:index])
			b = b[index:]
		}
		c, width := utf8.DecodeRune(b)
		b = b[width:]
		buf.WriteString(escapeRune(c))
	}
	buf.Write(b)
	buf.WriteRune('"')
}

func writeStringValue(buf *bytes.Buffer, s string) {
	if s == "" {
		buf.Write(bytesEmptyV)
		return
	}
	index := strings.IndexFunc(s, needsQuote)
	if index < 0 {
		buf.WriteString(s)
		return
	}
	buf.WriteRune('"')
	if index > 0 {
		buf.WriteString(s[0:index])
		s = s[index:]
	}
	for {
		index = strings.IndexFunc(s, needsBackslash)
		if index < 0 {
			break
		}
		if index > 0 {
			buf.WriteString(s[0:index])
			s = s[index:]
		}
		c, width := utf8.DecodeRuneInString(s)
		s = s[width:]
		buf.WriteString(escapeRune(c))
	}
	buf.WriteString(s)
	buf.WriteRune('"')
}

func writeTextMarshalerValue(buf *bytes.Buffer, t encoding.TextMarshaler) {
	b, err := t.MarshalText()
	if err != nil {
		buf.Write(bytesError)
		return
	}
	writeBytesValue(buf, b)
}

func needsQuote(c rune) bool {
	// This test will result in more values being quoted than is strictly
	// necesary for logfmt, but quoting all non-letter and non-digits makes
	// this compatible with the default colog extractor.
	return !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_'
}

func needsBackslash(c rune) bool {
	return c < ' ' || c == '\\' || c == '"'
}

func invalidKey(c rune) bool {
	return c <= ' ' || c == '=' || c == '"'
}

func escapeRune(c rune) string {
	if s, ok := escapeSequences[c]; ok {
		return s
	}
	return fmt.Sprintf(`\x%02x`, c)
}
