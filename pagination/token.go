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
// with any other byte is rejected by [Parse].
const tokenVersion = 1

// headerLen is the size of the envelope before the cursor payload: the
// version byte, the request-args checksum, and the cursor-shape checksum.
const headerLen = 1 + 8 + 8

// Token is a decoded page token. The zero value is a first page: it
// carries no cursor and mints tokens for the second page.
type Token struct {
	argsSum  uint64
	shapeSum uint64
	payload  []byte
}

// Parse decodes a request's page_token field. An empty token is a first
// page and always succeeds. The requestArgs are the request fields that
// must not change between pages, such as the parent, filter, and order_by;
// a token minted under different arguments fails with
// [ErrInvalidPageToken]. Pass page_size and skip through PageSize instead
// of including them here: AIP-158 allows them to vary between pages.
func Parse(token string, requestArgs ...any) (Token, error) {
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

// Cursor reports whether the token carries a cursor (false on the first
// page) and, when present, decodes it into dst, which must be a non-nil
// pointer to the same shape the cursor was minted from. A dst that is not
// a usable pointer fails with [ErrInvalidCursor]; a cursor minted from a
// different shape, or a corrupted payload, fails with
// [ErrInvalidPageToken].
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

// Next mints the token for the page after this one from the caller's
// cursor value, carrying over the request-args checksum the token was
// parsed with. The cursor may be any gob-encodable value, typically a
// small struct holding the ordering key of the page's last row; a value
// gob cannot encode fails with [ErrInvalidCursor].
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
