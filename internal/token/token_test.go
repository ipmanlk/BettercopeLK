package token

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []string{
		"https://zoom.lk/dunkirk-2017-sinhala-subtitle-%e0%b7%83%e0%b7%92%e0%b6%82%e0%b7%84%e0%b6%bd-%e0%b6%8b%e0%b6%b4%e0%b7%83%e0%b7%92%e0%b6%bb%e0%b7%90%e0%b7%83%e0%b7%92/",
		"https://baiscope.lk/movie-2021",
		"https://example.com/path/to/file?q=sinhala+%e0%b7%83&page=2",
		"https://example.com/",
	}
	for _, full := range tests {
		tok, err := EncodePath(full)
		if err != nil {
			t.Fatalf("EncodePath(%q): %v", full, err)
		}
		if strings.ContainsAny(tok, "+/=%") {
			t.Errorf("token %q contains URL-hostile characters", tok)
		}
		rel, err := DecodePath(tok)
		if err != nil {
			t.Fatalf("DecodePath(%q): %v", tok, err)
		}
		// Round-trip: re-encoding the decoded rel must equal the token.
		tok2, err := EncodePath("https://origin.example" + rel)
		if err != nil {
			t.Fatalf("re-encode: %v", err)
		}
		if tok2 != tok {
			t.Errorf("round trip mismatch: %q != %q", tok2, tok)
		}
	}
}

func TestEncodePathRejectsNonPath(t *testing.T) {
	for _, u := range []string{"not a url", "mailto:x@y.com"} {
		if _, err := EncodePath(u); err == nil {
			t.Errorf("expected error for %q", u)
		}
	}
}

func TestDecodePathRejectsInvalid(t *testing.T) {
	tests := []string{
		"",
		"%%%",
		"not-base64!!!",
		RawToken(t, "//evil.example.com/x"),      // protocol-relative
		RawToken(t, "/https://evil.example.com"), // scheme injection
		RawToken(t, "/a/../b"),                   // traversal
		RawToken(t, "/a\\..\\b"),                 // backslash traversal
		RawToken(t, "relative-no-slash"),         // no leading slash
	}
	for _, tok := range tests {
		if _, err := DecodePath(tok); err == nil {
			t.Errorf("expected error for token %q", tok)
		}
	}
}

func TestDecodePathRejectsOversize(t *testing.T) {
	big := "/" + strings.Repeat("a", maxTokenLen*2)
	tok := EncodeMust(t, big)
	if _, err := DecodePath(tok); err == nil {
		t.Error("expected error for oversized token")
	}
}

func TestDecodePathAcceptsQuery(t *testing.T) {
	rel := "/movie?q=%e0%b7%83%e0%b7%92&page=2"
	tok := EncodeMust(t, "https://example.com"+rel)
	got, err := DecodePath(tok)
	if err != nil {
		t.Fatalf("DecodePath: %v", err)
	}
	if got != rel {
		t.Errorf("got %q, want %q", got, rel)
	}
}

// RawToken base64-encodes rel directly, bypassing EncodePath validation, so
// tests can exercise invalid relative paths.
func RawToken(t *testing.T, rel string) string {
	t.Helper()
	return base64.RawURLEncoding.EncodeToString([]byte(rel))
}

// EncodeMust encodes a full URL to a token, failing the test on error.
func EncodeMust(t *testing.T, full string) string {
	t.Helper()
	tok, err := EncodePath(full)
	if err != nil {
		t.Fatalf("EncodePath(%q): %v", full, err)
	}
	return tok
}
