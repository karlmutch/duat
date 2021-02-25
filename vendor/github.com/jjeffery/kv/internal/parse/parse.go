// Package parse provides a parser for messages with key/value pairs.
package parse

import (
	"bytes"
	"sync"
)

const (
	// bufLength is the length of the buffer used for unquoting messages that
	// have one or more escape characters (\) in them. If a message reuires more
	// space than this, additional memory will need to be allocated.
	bufLength = 256

	// typicalKeywordCount is the number of keywords that can be in a
	// message before it becomes necessary to allocate additional memory.
	// The value is set to a number higher than expected in a typical
	// log message.
	typicalKeywordCount = 8
)

var (
	messagePool = sync.Pool{
		New: func() interface{} {
			return &Message{
				List: make([][]byte, 0, 2*typicalKeywordCount),
			}
		},
	}

	blankBuf [bufLength]byte
)

// Message represents a message with text and any assocated
// key/value pairs.
type Message struct {
	Text []byte          // message text
	List [][]byte        // key/value pairs
	buf  [bufLength]byte // for unquoting values
	used int             // number of bytes used in buf
}

func newMessage() *Message {
	return messagePool.Get().(*Message)
}

// Release returns the message to the pool for re-use.
func (m *Message) Release() {
	if m != nil {
		m.Text = nil
		for i := len(m.List) - 1; i >= 0; i-- {
			m.List[i] = nil
		}
		m.List = m.List[:0]
		if m.used > 0 {
			copy(m.buf[:m.used], blankBuf[:m.used])
			m.used = 0
		}
		messagePool.Put(m)
	}
}

// Bytes parses the input bytes and returns a message.
//
// Memory allocations are kept to a minimum. The message
// is allocated from a memory pool. Call Release()
// to return the message to the pool for re-use.
func Bytes(input []byte) *Message {
	lex := lexer{
		input: input,
	}
	lex.next()

	// firstKeyPos is the position of the first key in the message
	//
	// consider the following example message:
	//
	//  this is a message key=1 key=2 more message stuff key=3
	//                                                   ^
	// if a message has key=val and then text that       |
	// does not match key=val, then the key=val is       |
	// not parsed for example, the first key is here ----+
	var firstKeyPos int

	// count kv pairs so that we can allocate once only
	var kvCount int

	// iterate through the message looking for the position
	// before which we will not be looking for key/val pairs
	for lex.token != tokEOF {
		for lex.notMatch(tokKey, tokQuotedKey, tokEOF) {
			firstKeyPos = 0
			lex.next()
		}
		if lex.token == tokEOF {
			break
		}
		firstKeyPos = lex.pos
		for lex.match(tokKey, tokQuotedKey) {
			kvCount += 2
			lex.next() // skip past key
			lex.next() // skip past value
			lex.skipWS()
		}
	}

	lex.rewind()
	lex.skipWS()
	message := newMessage()
	unquoteBuf := message.buf[:]
	var unquoted []byte

	if firstKeyPos == 0 {
		// there are no key/value pairs
		message.Text = lex.input
	} else {
		if cap(message.List) < kvCount {
			message.List = make([][]byte, 0, kvCount)
		}
		var pos int
		for lex.pos < firstKeyPos {
			pos = lex.pos
			lex.next()
		}
		message.Text = lex.input[:pos]
		for lex.match(tokKey, tokQuotedKey) {
			if lex.token == tokKey {
				message.List = append(message.List, lex.lexeme())
			} else {
				unquoted, unquoteBuf = unquote(lex.lexeme(), unquoteBuf)
				message.List = append(message.List, unquoted)
			}
			lex.next()

			switch lex.token {
			case tokQuoted:
				unquoted, unquoteBuf = unquote(lex.lexeme(), unquoteBuf)
				message.List = append(message.List, unquoted)
			default:
				message.List = append(message.List, lex.lexeme())
			}

			lex.next()
			lex.skipWS()
		}
	}

	message.Text = bytes.TrimSpace(message.Text)
	message.used = bufLength - len(unquoteBuf)
	return message
}
