// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

// applyMarkReorder implements rule L3 in place on a visual-order permutation.
//
// After rule L2, a combining mark (Bidi_Class NSM) applied to a right-to-left
// base character precedes that base in visual order, because the base and its
// marks share an odd level and L2 reverses the whole run. L3 restores the
// base-then-marks order that shaping engines expect: each maximal group of one
// or more combining marks immediately followed, in visual order, by their base
// character (all at an odd level) is reversed.
//
// order is the L2 permutation of indices into text; levels are the per-rune
// levels of text. order is mutated in place.
func applyMarkReorder(text []rune, levels []Level, order []int) {
	i := 0
	for i < len(order) {
		if !isMarkAt(text, levels, order[i]) {
			i++
			continue
		}
		// A run of combining marks in visual order, all at an odd level.
		j := i
		for j < len(order) && isMarkAt(text, levels, order[j]) {
			j++
		}
		// The character following the marks is their base only when it too is
		// at an odd level; otherwise the marks have no right-to-left base here
		// and L3 does not apply to them.
		if j < len(order) && levels[order[j]]%2 == 1 {
			reverse(order[i : j+1])
			i = j + 1
			continue
		}
		i = j
	}
}

// isMarkAt reports whether the rune at logical index pos is a combining mark
// (Bidi_Class NSM) resolved to an odd (right-to-left) level.
func isMarkAt(text []rune, levels []Level, pos int) bool {
	return levels[pos]%2 == 1 && ClassOf(text[pos]) == NSM
}

// ReorderWithMarks runs rule L2 and then the rule L3 combining-mark refinement,
// returning the visual left-to-right order as a permutation of indices into
// text. It is the mark-aware counterpart of [Reorder]: the plain [Reorder]
// applies L2 only, matching the extent of the Unicode conformance data.
func ReorderWithMarks(text []rune, levels []Level) []int {
	order := reorderLevels(levels)
	applyMarkReorder(text, levels, order)
	return order
}
