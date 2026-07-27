// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleClass = `# DerivedBidiClass sample
# @missing: 0000..10FFFF; Left_To_Right
# @missing: 0590..05FF; Right_To_Left
0041; L # A
0030..0039; EN # digits
05D0..05EA; R # Hebrew
`

const sampleBrackets = `# BidiBrackets sample
0028; 0029; o # LEFT PARENTHESIS
0029; 0028; c # RIGHT PARENTHESIS
`

const sampleMirror = `# BidiMirroring sample
0028; 0029 # LEFT PARENTHESIS
0029; 0028 # RIGHT PARENTHESIS
`

const sampleShaping = `# ArabicShaping sample
0627; ALEF; R; ALEF
0628; BEH; D; BEH
0640; TATWEEL; C; NO_JOINING_GROUP
0600; NUMBER SIGN; U; No_Joining_Group
`

// fakeBody wraps a string as an http response body.
func fakeBody(s string) io.ReadCloser { return io.NopCloser(strings.NewReader(s)) }

// sequenceGetter returns an httpGet stub that yields the given responses in
// order.
func sequenceGetter(resps ...func() (*http.Response, error)) func(string) (*http.Response, error) {
	i := 0
	return func(string) (*http.Response, error) {
		r := resps[i]
		i++
		return r()
	}
}

func okResp(body string) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: fakeBody(body)}, nil
	}
}

func statusResp(code int) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		return &http.Response{StatusCode: code, Status: "404 Not Found", Body: fakeBody("")}, nil
	}
}

func errResp() func() (*http.Response, error) {
	return func() (*http.Response, error) { return nil, errors.New("boom") }
}

// withStubs installs test seams and restores them afterwards.
func withStubs(t *testing.T, get func(string) (*http.Response, error)) {
	t.Helper()
	oldGet, oldErr := httpGet, stderr
	httpGet = get
	stderr = io.Discard
	t.Cleanup(func() { httpGet, stderr = oldGet, oldErr })
}

// allOK returns the four success responses run() consumes in order.
func allOK() func(string) (*http.Response, error) {
	return sequenceGetter(okResp(sampleClass), okResp(sampleBrackets), okResp(sampleMirror), okResp(sampleShaping))
}

func TestRunSuccess(t *testing.T) {
	dir := t.TempDir()
	withStubs(t, allOK())
	if code := run([]string{"genbidi", dir}); code != 0 {
		t.Fatalf("run code=%d", code)
	}
	for _, name := range []string{
		"bidiclass_table.go", "bidibrackets_table.go",
		"bidimirror_table.go", "joining_table.go",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
}

func TestRunDefaultDir(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	withStubs(t, allOK())
	if code := run([]string{"genbidi"}); code != 0 {
		t.Fatalf("run code=%d", code)
	}
}

func TestRunErrors(t *testing.T) {
	dir := t.TempDir()

	t.Run("fetch class", func(t *testing.T) {
		withStubs(t, sequenceGetter(errResp()))
		if run([]string{"g", dir}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("parse class", func(t *testing.T) {
		withStubs(t, sequenceGetter(okResp("0041; NOPE\n")))
		if run([]string{"g", dir}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("write class", func(t *testing.T) {
		withStubs(t, sequenceGetter(okResp(sampleClass), okResp(sampleBrackets)))
		if run([]string{"g", filepath.Join(dir, "does-not-exist")}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("fetch brackets", func(t *testing.T) {
		withStubs(t, sequenceGetter(okResp(sampleClass), errResp()))
		if run([]string{"g", dir}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("parse brackets", func(t *testing.T) {
		withStubs(t, sequenceGetter(okResp(sampleClass), okResp("0028; 0029; x\n")))
		if run([]string{"g", dir}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("write brackets", func(t *testing.T) {
		bdir := t.TempDir()
		// Make the bracket output path un-writable by pre-creating a directory
		// with that name, while the class file still writes fine.
		if err := os.Mkdir(filepath.Join(bdir, "bidibrackets_table.go"), 0o755); err != nil {
			t.Fatal(err)
		}
		withStubs(t, sequenceGetter(okResp(sampleClass), okResp(sampleBrackets)))
		if run([]string{"g", bdir}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("fetch mirror", func(t *testing.T) {
		withStubs(t, sequenceGetter(okResp(sampleClass), okResp(sampleBrackets), errResp()))
		if run([]string{"g", dir}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("parse mirror", func(t *testing.T) {
		withStubs(t, sequenceGetter(okResp(sampleClass), okResp(sampleBrackets), okResp("0028\n")))
		if run([]string{"g", dir}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("write mirror", func(t *testing.T) {
		mdir := t.TempDir()
		if err := os.Mkdir(filepath.Join(mdir, "bidimirror_table.go"), 0o755); err != nil {
			t.Fatal(err)
		}
		withStubs(t, sequenceGetter(okResp(sampleClass), okResp(sampleBrackets), okResp(sampleMirror)))
		if run([]string{"g", mdir}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("fetch shaping", func(t *testing.T) {
		withStubs(t, sequenceGetter(okResp(sampleClass), okResp(sampleBrackets), okResp(sampleMirror), errResp()))
		if run([]string{"g", dir}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("parse shaping", func(t *testing.T) {
		withStubs(t, sequenceGetter(okResp(sampleClass), okResp(sampleBrackets), okResp(sampleMirror), okResp("0627; ALEF; Z; X\n")))
		if run([]string{"g", dir}) != 1 {
			t.Fatal("want 1")
		}
	})
	t.Run("write shaping", func(t *testing.T) {
		sdir := t.TempDir()
		if err := os.Mkdir(filepath.Join(sdir, "joining_table.go"), 0o755); err != nil {
			t.Fatal(err)
		}
		withStubs(t, allOK())
		if run([]string{"g", sdir}) != 1 {
			t.Fatal("want 1")
		}
	})
}

func TestFetch(t *testing.T) {
	withStubs(t, sequenceGetter(okResp("hello")))
	b, err := fetch("x")
	if err != nil || string(b) != "hello" {
		t.Fatalf("fetch=%q err=%v", b, err)
	}

	withStubs(t, sequenceGetter(errResp()))
	if _, err := fetch("x"); err == nil {
		t.Fatal("want transport error")
	}

	withStubs(t, sequenceGetter(statusResp(404)))
	if _, err := fetch("x"); err == nil {
		t.Fatal("want status error")
	}
}

func TestParse(t *testing.T) {
	ranges, err := parse([]byte(sampleClass))
	if err != nil {
		t.Fatal(err)
	}
	if len(ranges) == 0 {
		t.Fatal("no ranges")
	}
	// The default-ignorable missing line makes 0x0590 (unassigned) R.
	if got := classAt(ranges, 0x0590); got != "R" {
		t.Errorf("0x0590=%s want R", got)
	}
	if got := classAt(ranges, 0x0041); got != "L" {
		t.Errorf("0x0041=%s want L", got)
	}
	if got := classAt(ranges, 0x0035); got != "EN" {
		t.Errorf("0x0035=%s want EN", got)
	}

	if _, err := parse([]byte("0041; BOGUS\n")); err == nil {
		t.Error("want unknown-class error")
	}
	if _, err := parse([]byte("# @missing: 0041; BOGUS\n")); err == nil {
		t.Error("want unknown missing-class error")
	}
	if _, err := parse([]byte("ZZZZ; L\n")); err == nil {
		t.Error("want bad-hex error")
	}
}

func classAt(ranges []genRange, cp uint32) string {
	for _, r := range ranges {
		if cp >= r.lo && cp <= r.hi {
			return r.class
		}
	}
	return "?"
}

func TestParseAssignment(t *testing.T) {
	if _, _, _, err := parseAssignment("noSemicolon"); err == nil {
		t.Error("want missing-semicolon error")
	}
	if _, _, _, err := parseAssignment("ZZ..0030; L"); err == nil {
		t.Error("want bad lo-hex error")
	}
	if _, _, _, err := parseAssignment("0030..ZZ; L"); err == nil {
		t.Error("want bad hi-hex error")
	}
	if _, _, _, err := parseAssignment("ZZ; L"); err == nil {
		t.Error("want bad single-hex error")
	}
	if _, _, _, err := parseAssignment("0030; NOPE"); err == nil {
		t.Error("want unknown-class error")
	}
	lo, hi, _, err := parseAssignment("0030..0039; EN")
	if err != nil || lo != 0x30 || hi != 0x39 {
		t.Errorf("range parse: %x %x %v", lo, hi, err)
	}
}

func TestParseHex(t *testing.T) {
	if _, err := parseHex("110000"); err == nil {
		t.Error("want out-of-range error")
	}
	if _, err := parseHex("zz"); err == nil {
		t.Error("want parse error")
	}
	if v, err := parseHex(" 41 "); err != nil || v != 0x41 {
		t.Errorf("parseHex=%x err=%v", v, err)
	}
}

func TestMissingBody(t *testing.T) {
	if body, ok := missingBody("# @missing: 0041; L"); !ok || body != "0041; L" {
		t.Errorf("missingBody body=%q ok=%v", body, ok)
	}
	if _, ok := missingBody("0041; L"); ok {
		t.Error("non-missing line matched")
	}
}

func TestResolveClass(t *testing.T) {
	if c, err := resolveClass("AL"); err != nil || c != "AL" {
		t.Errorf("AL: %q %v", c, err)
	}
	if c, err := resolveClass("Arabic_Letter"); err != nil || c != "AL" {
		t.Errorf("Arabic_Letter: %q %v", c, err)
	}
	if _, err := resolveClass("Nope"); err == nil {
		t.Error("want unknown error")
	}
}

func TestParseBrackets(t *testing.T) {
	rows, err := parseBrackets([]byte(sampleBrackets))
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	if rows[0].kind != "bracketOpen" || rows[1].kind != "bracketClose" {
		t.Errorf("kinds: %+v", rows)
	}
	for _, bad := range []string{
		"0028; 0029\n",    // too few fields
		"ZZ; 0029; o\n",   // bad cp hex
		"0028; ZZ; o\n",   // bad pair hex
		"0028; 0029; x\n", // unknown kind
	} {
		if _, err := parseBrackets([]byte(bad)); err == nil {
			t.Errorf("want error for %q", bad)
		}
	}
}

func TestGenerate(t *testing.T) {
	src := generate([]genRange{{0, 0x40, "L"}, {0x41, 0x41, "R"}})
	if !strings.Contains(string(src), "package bidi") || !strings.Contains(string(src), "bidiRanges") {
		t.Error("generated class source malformed")
	}
	bsrc := generateBrackets([]bracketRow{{0x28, 0x29, "bracketOpen"}})
	if !strings.Contains(string(bsrc), "bracketData") {
		t.Error("generated bracket source malformed")
	}
	msrc := generateMirror([]mirrorRow{{0x28, 0x29}})
	if !strings.Contains(string(msrc), "mirrorData") || !strings.Contains(string(msrc), "BidiMirroring.txt") {
		t.Error("generated mirror source malformed")
	}
	ssrc := generateShaping([]shapeRow{{0x0628, "joinD"}})
	if !strings.Contains(string(ssrc), "joiningData") || !strings.Contains(string(ssrc), "ArabicShaping.txt") {
		t.Error("generated shaping source malformed")
	}
}

func TestParseMirror(t *testing.T) {
	rows, err := parseMirror([]byte(sampleMirror))
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	if rows[0].cp != 0x28 || rows[0].mirror != 0x29 {
		t.Errorf("row0=%+v", rows[0])
	}
	for _, bad := range []string{
		"0028\n",     // too few fields
		"ZZ; 0029\n", // bad cp hex
		"0028; ZZ\n", // bad mirror hex
	} {
		if _, err := parseMirror([]byte(bad)); err == nil {
			t.Errorf("want error for %q", bad)
		}
	}
}

func TestParseShaping(t *testing.T) {
	rows, err := parseShaping([]byte(sampleShaping))
	if err != nil {
		t.Fatal(err)
	}
	// The "U" (non-joining) row is dropped, leaving R, D and C.
	if len(rows) != 3 {
		t.Fatalf("rows=%d want 3", len(rows))
	}
	if rows[0].cp != 0x0627 || rows[0].kind != "joinR" {
		t.Errorf("row0=%+v", rows[0])
	}
	for _, bad := range []string{
		"0627; ALEF; R\n",       // too few fields
		"ZZ; ALEF; R; ALEF\n",   // bad cp hex
		"0627; ALEF; Z; ALEF\n", // unknown joining type
	} {
		if _, err := parseShaping([]byte(bad)); err == nil {
			t.Errorf("want error for %q", bad)
		}
	}
}

func TestMainFunc(t *testing.T) {
	dir := t.TempDir()
	oldExit, oldArgs := osExit, args
	code := -1
	osExit = func(c int) { code = c }
	args = []string{"genbidi", dir}
	withStubs(t, allOK())
	t.Cleanup(func() { osExit, args = oldExit, oldArgs })
	main()
	if code != 0 {
		t.Fatalf("main exit code=%d", code)
	}
}
