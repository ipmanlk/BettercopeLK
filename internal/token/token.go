// Package token encodes subtitle post-page URLs as compact, URL-safe tokens.
//
// The full URL of a search result is never sent to the browser. Instead the
// scraper encodes the path (and query) portion with URL-safe base64, and the
// frontend passes that opaque token back when requesting a download. This
// keeps payloads small and keeps the origin host entirely server-side.
package token

import (
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
)

// ErrInvalid is returned when a token cannot be decoded or fails validation.
var ErrInvalid = errors.New("invalid path token")

// maxTokenLen bounds the encoded token to prevent it being used as a storage
// or amplification channel.
const maxTokenLen = 4096

// EncodePath encodes the path and query of fullURL as a URL-safe base64 token.
// Only the request URI (path + query) is encoded; scheme and host stay server
// side.
func EncodePath(fullURL string) (string, error) {
	u, err := url.Parse(fullURL)
	if err != nil {
		return "", err
	}
	rel := u.RequestURI()
	if rel == "" || !strings.HasPrefix(rel, "/") {
		return "", ErrInvalid
	}
	return base64.RawURLEncoding.EncodeToString([]byte(rel)), nil
}

// DecodePath decodes tok back into the relative path it encodes, validating
// that the result is a safe, single-origin path.
func DecodePath(tok string) (string, error) {
	if tok == "" || len(tok) > maxTokenLen {
		return "", ErrInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		return "", ErrInvalid
	}
	rel := string(raw)
	if err := validateRel(rel); err != nil {
		return "", err
	}
	return rel, nil
}

// validateRel rejects relative paths that are absolute, protocol-relative, or
// contain traversal or control characters.
func validateRel(rel string) error {
	if rel == "" || !strings.HasPrefix(rel, "/") {
		return ErrInvalid
	}
	// Protocol-relative URLs (//host/path) would jump to another origin.
	if strings.HasPrefix(rel, "//") {
		return ErrInvalid
	}
	// Scheme injection, e.g. /https://evil.com.
	if strings.Contains(rel, "://") {
		return ErrInvalid
	}
	// Backslashes and control characters.
	for _, r := range rel {
		if r == '\\' || r < 0x20 {
			return ErrInvalid
		}
	}
	// Path traversal segments (../ and /../) are not allowed.
	pathPart := rel
	if i := strings.IndexByte(rel, '?'); i >= 0 {
		pathPart = rel[:i]
	}
	for _, seg := range strings.Split(pathPart, "/") {
		if seg == ".." {
			return ErrInvalid
		}
	}
	return nil
}
