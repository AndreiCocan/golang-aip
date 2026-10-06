package pagination

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"reflect"
)

// tokenVersion is the first byte of every minted token; a token starting
// with any other byte is rejected by [ParseToken].
const tokenVersion = 1

// headerLen is the size of the envelope before the cursor payload: the
// version byte, the request-args checksum, and the cursor-shape checksum.
const headerLen = 1 + 8 + 8

// Token is a decoded page token. The zero value is the first page of a
// request with no requestArgs: it has no cursor, and [Token.Next] makes the
// token of the second page.
type Token struct {
	argsSum  uint64
	shapeSum uint64
	payload  []byte
}

// ParseToken decodes the page_token field of a request. An empty token is a
// first page, and ParseToken then always succeeds.
//
// requestArgs are the request fields that must not change between pages,
// such as the parent, the filter, and the order_by. A token made with
// different arguments gives an error that matches [ErrInvalidPageToken].
// Do not put page_size or skip in requestArgs, because AIP-158 lets them
// change between pages. Give page_size to [ResolvePageSize].
//
// A malformed or damaged token also gives an error that matches
// [ErrInvalidPageToken].
func ParseToken(token string, requestArgs ...any) (Token, error) {
	argsSum := hashArgs(requestArgs)
	if token == "" {
		return Token{argsSum: argsSum}, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) <= headerLen || raw[0] != tokenVersion {
		return Token{}, ErrInvalidPageToken
	}

	if binary.BigEndian.Uint64(raw[1:9]) != argsSum {
		return Token{}, ErrInvalidPageToken
	}

	return Token{
		argsSum:  argsSum,
		shapeSum: binary.BigEndian.Uint64(raw[9:headerLen]),
		payload:  raw[headerLen:],
	}, nil
}

// Cursor decodes the cursor of the token into dst. It reports false, with
// no error, when the token has no cursor, which is the first page.
//
// dst must be a non-nil pointer to a value of the same shape as the cursor
// that [Token.Next] encoded. Cursor returns an error that matches:
//
//   - [ErrInvalidCursor] when dst is not a non-nil pointer, which is a bug in
//     the service.
//   - [ErrInvalidPageToken] when the cursor has a different shape than dst,
//     or when its data is damaged.
func (t Token) Cursor(dst any) (bool, error) {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return false, fmt.Errorf("%w: dst must be a non-nil pointer, got %T", ErrInvalidCursor, dst)
	}

	if len(t.payload) == 0 {
		return false, nil
	}

	if hashShape(v.Type().Elem()) != t.shapeSum {
		return false, ErrInvalidPageToken
	}

	if err := gob.NewDecoder(bytes.NewReader(t.payload)).Decode(dst); err != nil {
		return false, ErrInvalidPageToken
	}

	return true, nil
}

// Next returns the next_page_token: the token of the page after this one,
// with cursor in it. The token also holds the checksum of the requestArgs
// of [ParseToken], so that the next request must have the same arguments.
//
// cursor can be any value that gob can encode, usually a small struct with
// the sort key of the last row of the page. A nil cursor, or a value that
// gob cannot encode, gives an error that matches [ErrInvalidCursor].
func (t Token) Next(cursor any) (string, error) {
	v := reflect.ValueOf(cursor)
	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}

	if !v.IsValid() || v.Kind() == reflect.Pointer {
		return "", fmt.Errorf("%w: cursor must not be nil", ErrInvalidCursor)
	}

	var payload bytes.Buffer
	if err := gob.NewEncoder(&payload).Encode(v.Interface()); err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidCursor, err)
	}

	raw := make([]byte, 0, headerLen+payload.Len())
	raw = append(raw, tokenVersion)
	raw = binary.BigEndian.AppendUint64(raw, t.argsSum)
	raw = binary.BigEndian.AppendUint64(raw, hashShape(v.Type()))
	raw = append(raw, payload.Bytes()...)

	return base64.RawURLEncoding.EncodeToString(raw), nil
}
