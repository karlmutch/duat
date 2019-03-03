package parse

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

const (
	tokEOF = iota
	tokWord
	tokWS
	tokKey
	tokQuoted
	tokQuotedKey
)

var (
	errEOF = errors.New("eof")
)

type lexer struct {
	input []byte // input text, never modified
	start int    // start position of current lexeme
	end   int    // end position of current lexeme
	pos   int    // current position
	token int    // current token
}

func (lex *lexer) rewind() {
	lex.start = 0
	lex.end = 0
	lex.pos = 0
	lex.token = 0
	lex.next()
}

func (lex *lexer) lexeme() []byte {
	return lex.input[lex.start:lex.end]
}

func (lex *lexer) readRune() (rune, error) {
	ch, size := utf8.DecodeRune(lex.input[lex.pos:])
	if size == 0 {
		return 0, errEOF
	}
	if ch == utf8.RuneError {
		ch = '?'
	}
	lex.pos += size
	return ch, nil
}

func (lex *lexer) unreadRune() {
	if lex.pos > 0 {
		_, size := utf8.DecodeLastRune(lex.input[:lex.pos])
		lex.pos -= size
	}
}

func (lex *lexer) next() {
	lex.start = lex.pos
	ch, err := lex.readRune()
	if err != nil {
		lex.eof()
		return
	}
	if unicode.IsSpace(ch) {
		lex.whiteSpace(ch)
		return
	}
	if ch == '"' {
		lex.quoted(ch)
		return
	}

	lex.word(ch)
}

func (lex *lexer) eof() {
	lex.token = tokEOF
	lex.end = lex.pos
}

func (lex *lexer) whiteSpace(ch rune) {
	for {
		var err error
		ch, err = lex.readRune()
		if err != nil {
			break
		}
		if !unicode.IsSpace(ch) {
			lex.unreadRune()
			break
		}
	}
	lex.token = tokWS
	lex.end = lex.pos
}

func (lex *lexer) quoted(quote rune) {
	lex.token = tokQuoted
	var escaped bool
	for {
		ch, err := lex.readRune()
		if err != nil {
			lex.end = lex.pos
			return
		}
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == quote {
			break
		}
	}

	lex.end = lex.pos

	// lose any ":" separator after a quoted value
	ch, err := lex.readRune()
	if err == nil {
		switch ch {
		case ':':
			// remove any ':' separator after a quoted value
			break
		case '=':
			// an equals at the end of a quoted value means treat
			// it as a keyword
			lex.token = tokQuotedKey
		default:
			lex.unreadRune()
		}
	}
}

func (lex *lexer) word(ch rune) {
	token := tokWord
loop:
	for {
		var err error
		lex.end = lex.pos
		ch, err = lex.readRune()
		if err != nil {
			break
		}
		if unicode.IsSpace(ch) {
			lex.unreadRune()
			break
		}
		if ch == '=' {
			// Only consider this a keyword if the next character
			// after the equals is a non-space character. This picks
			// up cases where, for example, a base64 value is logged
			// that has one or more '=' chars at the end.
			ch, err = lex.readRune()
			if err != nil {
				// eof, so the equals is just part of the word
				lex.end = lex.pos
				break
			}
			if unicode.IsSpace(ch) {
				// equals is part of the word
				lex.unreadRune()
				lex.end = lex.pos
				break
			}
			if ch == '=' {
				// more than one terminating equals, can happen
				// with base64, base32 style encoding
				for {
					ch, err = lex.readRune()
					if err != nil {
						lex.end = lex.pos
						break loop
					}
					if unicode.IsSpace(ch) {
						lex.unreadRune()
						lex.end = lex.pos
						break loop
					}
				}
			}

			// Next char is non-space, non-equals, so we consider
			// this to be a key. Note that end is still pointing
			// to the beginning of the '=', so it is not part
			// of the lexeme.
			lex.unreadRune()
			token = tokKey
			break
		}
		if lex.token == tokKey || lex.token == tokQuotedKey {
			// unquoted colon terminates a value
			if ch == ':' {
				break
			}
		}
	}
	lex.token = token
	return
}

func (lex *lexer) skipWS() {
	for lex.token == tokWS {
		lex.next()
	}
}

func (lex *lexer) notMatch(toks ...int) bool {
	for _, tok := range toks {
		if tok == lex.token {
			return false
		}
	}
	return true
}

func (lex *lexer) match(toks ...int) bool {
	for _, tok := range toks {
		if tok == lex.token {
			return true
		}
	}
	return false
}
