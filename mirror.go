// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

// Mirror implements the character half of rule L4: it returns the code point of
// the character whose glyph mirrors r (the Bidi_Mirroring_Glyph property), or r
// itself when r has no mirrored counterpart.
//
// Mirror is unconditional; it does not consult an embedding level. Callers that
// only want to mirror characters resolved to a right-to-left level should use
// [MirrorRunes], or guard the call with the character's [Level].
func Mirror(r rune) rune {
	if m, ok := mirrorData[r]; ok {
		return m
	}
	return r
}

// MirrorRunes implements rule L4 over a paragraph: it returns a copy of text in
// which every character resolved to an odd (right-to-left) embedding level is
// replaced by its mirrored glyph (see [Mirror]); characters at even levels, and
// characters without a mirror, are copied unchanged.
//
// levels must be the per-rune levels of text (as returned by [ResolveLevels]);
// a rune with no corresponding level is treated as left-to-right.
func MirrorRunes(text []rune, levels []Level) []rune {
	out := make([]rune, len(text))
	for i, r := range text {
		if i < len(levels) && levels[i]%2 == 1 {
			out[i] = Mirror(r)
		} else {
			out[i] = r
		}
	}
	return out
}
