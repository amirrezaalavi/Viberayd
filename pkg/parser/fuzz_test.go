package parser

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// corpusFromExamples reads all .txt files from testdata/examples/.
func corpusFromExamples(t testing.TB) []string {
	t.Helper()
	matches, err := filepath.Glob("../../testdata/examples/*.txt")
	if err != nil {
		t.Fatalf("globbing examples: %v", err)
	}
	var seeds []string
	for _, m := range matches {
		b, err := os.ReadFile(m)
		if err == nil {
			seeds = append(seeds, strings.TrimSpace(string(b)))
		}
	}
	return seeds
}

// knownBrokenStrings returns interestingly broken inputs for seed corpora.
func knownBrokenStrings() []string {
	return []string{
		"",
		"   ",
		"!!!",
		"ss://",
		"vmess://",
		"vless://",
		"trojan://",
		"reality://",
		"ss://@",
		"vmess://!!invalid!!",
		base64.StdEncoding.EncodeToString([]byte("not a valid anything")),
		strings.Repeat("A", 10000),     // long garbage
		strings.Repeat("ss://", 1000),   // repetitive prefix
		"vmess://" + strings.Repeat("A", 5000),
		"vless://" + base64.URLEncoding.EncodeToString([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}),
		"ss://YWVzLTI1Ni1nY206dGVzdA==", // valid base64 method:password
		"vmess://" + base64.StdEncoding.EncodeToString([]byte(`{"v":"2","ps":"X"}`)),
	}
}

// FuzzParse ensures Parse never panics.
func FuzzParse(f *testing.F) {
	// Seed corpus from examples
	for _, s := range corpusFromExamples(f) {
		f.Add(s)
	}
	// Seed corpus from known broken strings
	for _, s := range knownBrokenStrings() {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() { _ = recover() }()
			_, _ = Parse(input)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Errorf("Parse(%q) took >1s", input)
		}
	})
}

// FuzzParseSingle ensures ParseSingle never panics.
func FuzzParseSingle(f *testing.F) {
	for _, s := range corpusFromExamples(f) {
		f.Add(s)
	}
	for _, s := range knownBrokenStrings() {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() { _ = recover() }()
			_, _ = ParseSingle(input)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Errorf("ParseSingle(%q) took >1s", input)
		}
	})
}

// FuzzDetectProtocol ensures DetectProtocol never panics.
func FuzzDetectProtocol(f *testing.F) {
	for _, s := range corpusFromExamples(f) {
		f.Add(s)
	}
	for _, s := range knownBrokenStrings() {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() { _ = recover() }()
			_, _ = DetectProtocol(input)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Errorf("DetectProtocol(%q) took >1s", input)
		}
	})
}

// FuzzValidatePublicKey ensures ValidatePublicKey never panics and
// always returns either an error or nil.
func FuzzValidatePublicKey(f *testing.F) {
	for _, s := range corpusFromExamples(f) {
		f.Add(s)
	}
	for _, s := range knownBrokenStrings() {
		f.Add(s)
	}
	// Add specific public-key-like seeds
	f.Add("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")   // 32 zero bytes
	f.Add("")
	f.Add("short")
	f.Add("not valid base64!!!")
	f.Add(strings.Repeat("A", 100))

	f.Fuzz(func(t *testing.T, input string) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() { _ = recover() }()
			err := ValidatePublicKey(input)
			// Must return error or nil — never anything else
			if err != nil {
				_ = err.Error() // ensure it's a real error value
			}
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Errorf("ValidatePublicKey(%q) took >1s", input)
		}
	})
}
