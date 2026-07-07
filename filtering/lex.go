package filtering

import (
	"fmt"
	"strings"
)

// tokenKind identifies a lexical token of the filter grammar.
type tokenKind uint8

const (
	tokenEOF tokenKind = iota
	tokenText
	tokenString
	tokenDot
	tokenComma
	tokenLparen
	tokenRparen
	tokenMinus
	tokenEquals
	tokenNotEquals
	tokenLess
	tokenLessEquals
	tokenGreater
	tokenGreaterEquals
	tokenHas
)

// token is one lexical token. pos and end are byte offsets into the filter:
// filter[pos:end] is the token as written, including quotes for a string.
type token struct {
	kind tokenKind
	pos  int
	end  int
	// text is the word of a TEXT token or the unquoted content of a STRING
	// token, and empty for operator tokens.
	text string
	// spaceBefore reports whether whitespace immediately precedes the
	// token. The grammar needs it to separate sequence factors and to
	// validate keyword spacing.
	spaceBefore bool
}

// lastControlByte is the highest ASCII control character; bytes above it
// are printable ASCII or part of a multi-byte UTF-8 rune.
const lastControlByte = 0x1f

// isTextByte reports whether c may appear inside a TEXT token. TEXT stops
// at whitespace, structural punctuation, comparator characters, quotes, and
// control characters; everything else, including "-", "*", and all
// non-ASCII bytes, is text.
func isTextByte(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '(', ')', ',', '.', '<', '>', '=', '!', ':', '"', '\'':
		return false
	}

	return c > lastControlByte
}

// lex splits filter into tokens, ending with a tokenEOF token. It fails with
// a [*ParseError] on an unterminated string or a stray "!".
func lex(filter string) ([]token, error) {
	var (
		tokens []token
		space  bool
	)

	emit := func(kind tokenKind, pos, end int, text string) {
		tokens = append(tokens, token{
			kind:        kind,
			pos:         pos,
			end:         end,
			text:        text,
			spaceBefore: space,
		})
		space = false
	}
	fail := func(pos int, msg string) ([]token, error) {
		return nil, &ParseError{Filter: filter, Pos: pos, Message: msg}
	}

	i := 0
	for i < len(filter) {
		pos := i
		switch c := filter[i]; c {
		case ' ', '\t', '\n', '\r':
			i++
			space = true
		case '(':
			i++
			emit(tokenLparen, pos, i, "")
		case ')':
			i++
			emit(tokenRparen, pos, i, "")
		case ',':
			i++
			emit(tokenComma, pos, i, "")
		case '.':
			i++
			emit(tokenDot, pos, i, "")
		case ':':
			i++
			emit(tokenHas, pos, i, "")
		case '=':
			i++
			emit(tokenEquals, pos, i, "")
		case '!':
			if i+1 >= len(filter) || filter[i+1] != '=' {
				return fail(pos, `"!" must be followed by "="`)
			}

			i += 2
			emit(tokenNotEquals, pos, i, "")
		case '<':
			if i+1 < len(filter) && filter[i+1] == '=' {
				i += 2
				emit(tokenLessEquals, pos, i, "")
			} else {
				i++
				emit(tokenLess, pos, i, "")
			}
		case '>':
			if i+1 < len(filter) && filter[i+1] == '=' {
				i += 2
				emit(tokenGreaterEquals, pos, i, "")
			} else {
				i++
				emit(tokenGreater, pos, i, "")
			}
		case '-':
			i++
			emit(tokenMinus, pos, i, "")
		case '"', '\'':
			content := strings.IndexByte(filter[i+1:], c)
			if content == -1 {
				return fail(pos, "unterminated string")
			}

			i += 1 + content + 1
			emit(tokenString, pos, i, filter[pos+1:i-1])
		default:
			for i < len(filter) && isTextByte(filter[i]) {
				i++
			}

			if i == pos {
				return fail(pos, fmt.Sprintf("unexpected character %q", rune(c)))
			}

			emit(tokenText, pos, i, filter[pos:i])
		}
	}

	emit(tokenEOF, len(filter), len(filter), "")

	return tokens, nil
}
