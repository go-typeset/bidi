// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi_test

import (
	"fmt"

	"github.com/go-opentype/bidi"
)

// ExampleVisualOrder reorders a logical-order string mixing English (L) and
// Hebrew (R) into left-to-right visual order, the one-call convenience most
// callers want.
func ExampleVisualOrder() {
	s := "abc אבג def"
	fmt.Println(bidi.VisualOrder(s, bidi.Auto))
	// Output: abc גבא def
}

// ExampleResolveLevels shows the lower-level API VisualOrder builds on:
// ResolveLevels assigns an embedding Level to every rune (even = LTR, odd =
// RTL), and Reorder turns those levels into a visual-order index
// permutation.
func ExampleResolveLevels() {
	text := []rune("abc אבג def")
	levels := bidi.ResolveLevels(text, bidi.LeftToRight)
	order := bidi.Reorder(text, levels)

	fmt.Println(levels)
	fmt.Println(order)
	// Output:
	// [0 0 0 0 1 1 1 0 0 0 0]
	// [0 1 2 3 6 5 4 7 8 9 10]
}

// ExampleClassOf reports the Unicode Bidi_Class of a few runes: a Hebrew
// letter is strong right-to-left (R), a Latin letter is strong left-to-right
// (L), and a digit is a European number (EN).
func ExampleClassOf() {
	fmt.Println(bidi.ClassOf('א'))
	fmt.Println(bidi.ClassOf('A'))
	fmt.Println(bidi.ClassOf('1'))
	// Output:
	// R
	// L
	// EN
}

// ExampleMirror applies rule L4: a paired punctuation mark gets its mirrored
// glyph when it will be displayed at a right-to-left level, while a letter
// like Hebrew alef (which has no mirrored counterpart) is returned
// unchanged.
func ExampleMirror() {
	fmt.Printf("%c\n", bidi.Mirror('('))
	fmt.Println(bidi.Mirror('א') == 'א')
	// Output:
	// )
	// true
}

// ExampleJoinForms resolves the Arabic cursive presentation form of each
// letter in a word: an initial-joining letter, a dual-joining letter taking
// its medial form, a right-joining-only letter taking its final form, and a
// letter that starts a fresh join (isolated) because its predecessor does
// not connect forward.
func ExampleJoinForms() {
	names := map[bidi.JoinForm]string{
		bidi.Isolated: "Isolated",
		bidi.Initial:  "Initial",
		bidi.Medial:   "Medial",
		bidi.Final:    "Final",
	}
	word := []rune("سلام") // "peace"
	for i, form := range bidi.JoinForms(word) {
		fmt.Printf("%c: %s\n", word[i], names[form])
	}
	// Output:
	// س: Initial
	// ل: Medial
	// ا: Final
	// م: Isolated
}

// ExampleVisualParagraphs runs the full display pipeline (split into
// paragraphs, resolve, reorder) over multi-paragraph text in one call.
func ExampleVisualParagraphs() {
	for _, p := range bidi.VisualParagraphs("abc\nאבג", bidi.Auto) {
		fmt.Printf("%q\n", p)
	}
	// Output:
	// "abc\n"
	// "גבא"
}
