// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

import "testing"

// Representative Arabic code points used by the joining tests.
const (
	rLam      = 0x0644 // LAM,  dual-joining
	rAlef     = 0x0627 // ALEF, right-joining
	rBeh      = 0x0628 // BEH,  dual-joining
	rTatweel  = 0x0640 // TATWEEL, join-causing
	rFathatan = 0x064B // FATHATAN, transparent
)

// TestMirror covers the rune-level rule L4 mapping, including the no-mirror
// passthrough.
func TestMirror(t *testing.T) {
	if got := Mirror(rParenL); got != rParenR {
		t.Errorf("Mirror('(')=%#x want ')'", got)
	}
	if got := Mirror(rParenR); got != rParenL {
		t.Errorf("Mirror(')')=%#x want '('", got)
	}
	if got := Mirror(rLatinA); got != rLatinA {
		t.Errorf("Mirror('A')=%#x want unchanged", got)
	}
}

// TestMirrorRunes exercises the level-aware L4 application: mirrored at odd
// levels, unchanged at even levels, and unchanged where no level is provided.
func TestMirrorRunes(t *testing.T) {
	text := []rune{rParenL, rParenL, rParenL}
	// Third rune has no matching level → treated as left-to-right (unchanged).
	levels := []Level{1, 0}
	got := MirrorRunes(text, levels)
	want := []rune{rParenR, rParenL, rParenL}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("MirrorRunes[%d]=%#x want %#x", i, got[i], want[i])
		}
	}
}

// TestParagraphs covers rule P1 splitting: plain separators, CR+LF as one
// separator, a lone CR, a CR not followed by LF, and the no-trailing-empty case.
func TestParagraphs(t *testing.T) {
	cases := []struct {
		name string
		in   []rune
		want []string
	}{
		{"none", []rune("abc"), []string{"abc"}},
		{"lf", []rune("a\nb"), []string{"a\n", "b"}},
		{"crlf", []rune("a\r\nb"), []string{"a\r\n", "b"}},
		{"cr-only", []rune("a\r"), []string{"a\r"}},
		{"cr-then-text", []rune("a\rb"), []string{"a\r", "b"}},
		{"trailing-sep", []rune("a\n"), []string{"a\n"}},
		{"double-sep", []rune("a\n\nb"), []string{"a\n", "\n", "b"}},
		{"empty", nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Paragraphs(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("len=%d want %d (%q)", len(got), len(c.want), c.in)
			}
			for i := range c.want {
				if string(got[i]) != c.want[i] {
					t.Errorf("para %d=%q want %q", i, string(got[i]), c.want[i])
				}
			}
		})
	}
}

// TestVisualParagraphs runs the full per-paragraph pipeline: multiple
// paragraphs, RTL mirroring, X9-removed characters dropped, and the empty case.
func TestVisualParagraphs(t *testing.T) {
	// Empty input yields no paragraphs.
	if got := VisualParagraphs("", Auto); len(got) != 0 {
		t.Fatalf("empty: len=%d", len(got))
	}

	// Single RTL paragraph: the parenthesis is at an odd level and is mirrored,
	// and the run is reversed to visual order.
	rtl := VisualParagraphs(string([]rune{rHebAlef, rParenL}), RightToLeft)
	if len(rtl) != 1 || rtl[0] != string([]rune{rParenR, rHebAlef}) {
		t.Errorf("RTL mirror: %#v", []rune(rtl[0]))
	}

	// Two LTR paragraphs; the second contains removed formatting characters
	// that must be dropped from the visual string.
	multi := VisualParagraphs(string([]rune{rLatinA, rPara, rLatinA, rLRE, rLatinA + 1, rPDF}), LeftToRight)
	if len(multi) != 2 {
		t.Fatalf("multi: len=%d want 2", len(multi))
	}
	if multi[1] != "AB" {
		t.Errorf("removed not dropped: %q", multi[1])
	}
}

// TestReorderWithMarks covers rule L3: a combining mark on a right-to-left base
// is placed after its base in visual order, while plain L2 leaves it before.
func TestReorderWithMarks(t *testing.T) {
	text := []rune{rHebAlef, rGrave} // base + combining mark, RTL
	levels := ResolveLevels(text, RightToLeft)

	if l2 := Reorder(text, levels); !equalInts(l2, []int{1, 0}) {
		t.Errorf("L2 order=%v want [1 0]", l2)
	}
	if l3 := ReorderWithMarks(text, levels); !equalInts(l3, []int{0, 1}) {
		t.Errorf("L3 order=%v want [0 1]", l3)
	}

	// A left-to-right run needs no mark reordering (marks already follow base).
	ltr := []rune{rLatinA, rLatinA + 1}
	ll := ResolveLevels(ltr, LeftToRight)
	if got := ReorderWithMarks(ltr, ll); !equalInts(got, []int{0, 1}) {
		t.Errorf("LTR order=%v want [0 1]", got)
	}

	// A trailing combining mark with no right-to-left base after it is left in
	// place (the L3 else branch): a lone NSM in an RTL paragraph.
	lone := []rune{rGrave}
	lv := ResolveLevels(lone, RightToLeft)
	if got := ReorderWithMarks(lone, lv); !equalInts(got, []int{0}) {
		t.Errorf("lone mark order=%v want [0]", got)
	}
}

// TestJoinForms covers the Arabic joining algorithm: isolated, initial/medial/
// final progression, the LAM+ALEF context, join-causing TATWEEL, transparent
// marks, and the non-joining default.
func TestJoinForms(t *testing.T) {
	eq := func(name string, got, want []JoinForm) {
		if len(got) != len(want) {
			t.Fatalf("%s: len=%d want %d", name, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: form[%d]=%d want %d", name, i, got[i], want[i])
			}
		}
	}

	eq("isolated", JoinForms([]rune{rBeh}), []JoinForm{Isolated})
	eq("lam-alef", JoinForms([]rune{rLam, rAlef}), []JoinForm{Initial, Final})
	eq("triple", JoinForms([]rune{rBeh, rBeh, rBeh}), []JoinForm{Initial, Medial, Final})
	eq("tatweel", JoinForms([]rune{rBeh, rTatweel, rBeh}), []JoinForm{Initial, Medial, Final})
	// A transparent mark between two dual-joining letters keeps them joined and
	// is itself reported as Isolated.
	eq("transparent", JoinForms([]rune{rBeh, rFathatan, rBeh}), []JoinForm{Initial, Isolated, Final})
	// A non-joining (default) character.
	eq("non-joining", JoinForms([]rune{rLatinA}), []JoinForm{Isolated})
}

// TestPresentationForm covers the Arabic Presentation Forms-B fallback mapping,
// including a letter lacking the requested form, an unmapped rune, and an
// out-of-range form value.
func TestPresentationForm(t *testing.T) {
	if got := PresentationForm(rBeh, Initial); got != 0xFE91 {
		t.Errorf("BEH initial=%#x want FE91", got)
	}
	if got := PresentationForm(rBeh, Isolated); got != 0xFE8F {
		t.Errorf("BEH isolated=%#x want FE8F", got)
	}
	// ALEF is right-joining: it has no initial form, so r is returned.
	if got := PresentationForm(rAlef, Initial); got != rAlef {
		t.Errorf("ALEF initial=%#x want unchanged", got)
	}
	// Unmapped rune.
	if got := PresentationForm(rLatinA, Isolated); got != rLatinA {
		t.Errorf("A=%#x want unchanged", got)
	}
	// Out-of-range form value.
	if got := PresentationForm(rBeh, JoinForm(99)); got != rBeh {
		t.Errorf("bad form=%#x want unchanged", got)
	}
}
