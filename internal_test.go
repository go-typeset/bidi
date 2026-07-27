// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

import "testing"

// Representative code points used across the corner-case tests.
const (
	rLatinA  = 0x0041 // 'A'      L
	rHebAlef = 0x05D0 // Hebrew   R
	rAraAlef = 0x0627 // Arabic   AL
	rDigit0  = 0x0030 // '0'      EN
	rDollar  = 0x0024 // '$'      ET
	rGrave   = 0x0300 // combine  NSM
	rSpace   = 0x0020 // ' '      WS
	rLRE     = 0x202A
	rRLE     = 0x202B
	rPDF     = 0x202C
	rLRI     = 0x2066
	rRLI     = 0x2067
	rFSI     = 0x2068
	rPDI     = 0x2069
	rPara    = 0x2029 // PARAGRAPH SEPARATOR, class B
	rParenL  = 0x0028
	rParenR  = 0x0029
	rAngL    = 0x2329 // canonically equivalent to 0x3008
	rAngR    = 0x232A // canonically equivalent to 0x3009
)

func TestClassOf(t *testing.T) {
	cases := []struct {
		r    rune
		want Class
	}{
		{rLatinA, L},
		{rHebAlef, R},
		{rAraAlef, AL},
		{rDigit0, EN},
		{rDollar, ET},
		{rGrave, NSM},
		{rSpace, WS},
		{rLRI, LRI},
		{rPDI, PDI},
		{rPara, B},
		{0x0660, AN},  // Arabic-Indic digit zero
		{0x000A, B},   // line feed
		{-1, L},       // out of range → default L
		{0x110000, L}, // above max rune → default L
		{0x05EB, R},   // unassigned in Hebrew block → default R
	}
	for _, c := range cases {
		if got := ClassOf(c.r); got != c.want {
			t.Errorf("ClassOf(%#x)=%v want %v", c.r, got, c.want)
		}
	}
}

func TestClassString(t *testing.T) {
	if got := L.String(); got != "L" {
		t.Errorf("L.String()=%q", got)
	}
	if got := AL.String(); got != "AL" {
		t.Errorf("AL.String()=%q", got)
	}
	if got := Class(200).String(); got != "Class(200)" {
		t.Errorf("Class(200).String()=%q", got)
	}
	if got := itoa(0); got != "0" {
		t.Errorf("itoa(0)=%q", got)
	}
	if got := itoa(4090); got != "4090" {
		t.Errorf("itoa(4090)=%q", got)
	}
}

func TestCanonBracket(t *testing.T) {
	if canonBracket(rAngL) != 0x3008 {
		t.Error("2329 should canonicalize to 3008")
	}
	if canonBracket(rAngR) != 0x3009 {
		t.Error("232A should canonicalize to 3009")
	}
	if canonBracket(rParenL) != rParenL {
		t.Error("parenthesis should be unchanged")
	}
}

func TestVisualOrder(t *testing.T) {
	if got := VisualOrder("abc", LeftToRight); got != "abc" {
		t.Errorf("LTR ascii: %q", got)
	}
	// Two Hebrew letters read right-to-left.
	heb := string([]rune{rHebAlef, rHebAlef + 1})
	if got := VisualOrder(heb, RightToLeft); got != string([]rune{rHebAlef + 1, rHebAlef}) {
		t.Errorf("RTL hebrew: %#v", []rune(got))
	}
	// A removed formatting character (LRE/PDF) is dropped from the output.
	in := string([]rune{rLatinA, rLRE, rLatinA + 1, rPDF})
	if got := VisualOrder(in, LeftToRight); got != "AB" {
		t.Errorf("dropped formatting: %q", got)
	}
	if got := VisualOrder("", Auto); got != "" {
		t.Errorf("empty: %q", got)
	}
}

func TestReorderPublic(t *testing.T) {
	runes := []rune{rHebAlef, rHebAlef + 1, rHebAlef + 2}
	levels := ResolveLevels(runes, RightToLeft)
	order := Reorder(runes, levels)
	want := []int{2, 1, 0}
	if !equalInts(order, want) {
		t.Errorf("order=%v want %v", order, want)
	}
}

// TestAutoBaseLevel exercises P2/P3 including isolate-skipping in firstStrong.
func TestAutoBaseLevel(t *testing.T) {
	// Isolate content is skipped; the strong R after the PDI wins.
	in := []rune{rLRI, rLatinA, rPDI, rHebAlef}
	if got := BaseLevel(in, Auto); got != 1 {
		t.Errorf("auto with skipped isolate: level=%d want 1", got)
	}
	// No strong character at all → default LTR.
	if got := BaseLevel([]rune{rSpace, rDigit0}, Auto); got != 0 {
		t.Errorf("auto no-strong: level=%d want 0", got)
	}
	// First strong is L.
	if got := BaseLevel([]rune{rLatinA, rHebAlef}, Auto); got != 0 {
		t.Errorf("auto first-strong-L: level=%d want 0", got)
	}
}

// TestFSIDirections covers FSI resolving to both RLI and LRI.
func TestFSIDirections(t *testing.T) {
	// FSI whose first strong content is R behaves like RLI.
	rtl := []rune{rFSI, rHebAlef, rPDI}
	lv := ResolveLevels(rtl, LeftToRight)
	if lv[1]%2 != 1 {
		t.Errorf("FSI(R) inner level=%d, want odd", lv[1])
	}
	// FSI whose first strong content is L behaves like LRI.
	ltr := []rune{rFSI, rLatinA, rPDI}
	lv = ResolveLevels(ltr, RightToLeft)
	if lv[1]%2 != 0 {
		t.Errorf("FSI(L) inner level=%d, want even", lv[1])
	}
}

// TestParagraphSeparator exercises rule X8 (class B mid-text).
func TestParagraphSeparator(t *testing.T) {
	in := []rune{rHebAlef, rPara, rLatinA}
	lv := ResolveLevels(in, LeftToRight)
	if lv[1] != 0 {
		t.Errorf("B level=%d want paragraph level 0", lv[1])
	}
}

// TestEmbeddingOverflow drives explicit embeddings past maxDepth to cover the
// overflow-embedding and PDF-pop branches of rule X.
func TestEmbeddingOverflow(t *testing.T) {
	var in []rune
	for i := 0; i < 130; i++ {
		in = append(in, rRLE)
	}
	in = append(in, rLatinA)
	for i := 0; i < 130; i++ {
		in = append(in, rPDF)
	}
	// A lone PDF with nothing to pop must be a no-op, not a panic.
	in = append(in, rPDF, rLatinA)
	if lv := ResolveLevels(in, LeftToRight); len(lv) != len(in) {
		t.Fatalf("length mismatch")
	}
}

// TestIsolateOverflow drives isolates past maxDepth and pops embeddings that
// are still open when a PDI arrives.
func TestIsolateOverflow(t *testing.T) {
	var in []rune
	for i := 0; i < 130; i++ {
		in = append(in, rRLI)
	}
	in = append(in, rLatinA)
	for i := 0; i < 130; i++ {
		in = append(in, rPDI)
	}
	if lv := ResolveLevels(in, LeftToRight); len(lv) != len(in) {
		t.Fatal("length mismatch")
	}
	// PDI that pops an embedding still open inside its isolate.
	in2 := []rune{rRLI, rLRE, rLatinA, rPDI}
	if lv := ResolveLevels(in2, LeftToRight); len(lv) != 4 {
		t.Fatal("length mismatch")
	}
	// Stray PDI with no matching isolate is ignored.
	in3 := []rune{rPDI, rLatinA}
	if lv := ResolveLevels(in3, LeftToRight); len(lv) != 2 {
		t.Fatal("length mismatch")
	}
}

// TestWeakRules exercises W2 (EN→AN after AL), W5 (ET adjacent to EN) and the
// W1 NSM-after-PDI case.
func TestWeakRules(t *testing.T) {
	// AL followed by EN: W2 turns EN into AN, W3 turns AL into R.
	if lv := ResolveLevels([]rune{rAraAlef, rDigit0}, LeftToRight); len(lv) != 2 {
		t.Fatal("w2 length")
	}
	// ET adjacent to EN becomes EN (W5), both leading and trailing.
	ResolveLevels([]rune{rDollar, rDigit0}, LeftToRight)
	ResolveLevels([]rune{rDigit0, rDollar}, LeftToRight)
	// NSM immediately after a PDI resolves to ON (W1).
	ResolveLevels([]rune{rRLI, rHebAlef, rPDI, rGrave}, LeftToRight)
}

// TestBracketStackOverflow forces more than 63 nested opening brackets so BD16
// abandons processing.
func TestBracketStackOverflow(t *testing.T) {
	var in []rune
	in = append(in, rHebAlef)
	for i := 0; i < 70; i++ {
		in = append(in, rParenL)
	}
	in = append(in, rLatinA)
	for i := 0; i < 70; i++ {
		in = append(in, rParenR)
	}
	if lv := ResolveLevels(in, RightToLeft); len(lv) != len(in) {
		t.Fatal("length mismatch")
	}
}

// TestBracketCanonicalPair pairs U+2329 with U+3009 via canonical equivalence
// inside an RTL context so N0 resolves the pair.
func TestBracketCanonicalPair(t *testing.T) {
	in := []rune{rHebAlef, rAngL, rHebAlef, 0x3009}
	if lv := ResolveLevels(in, RightToLeft); len(lv) != 4 {
		t.Fatal("length mismatch")
	}
}
