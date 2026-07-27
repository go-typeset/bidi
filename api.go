// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

// applyL1 implements rule L1: segment and paragraph separators, and any run of
// whitespace or isolate formatting characters preceding them or ending the
// paragraph, are reset to the paragraph embedding level. Original types are
// used, per the rule.
func (p *paragraph) applyL1() {
	n := len(p.orig)
	reset := func(end int) {
		for j := end; j >= 0; j-- {
			c := p.orig[j]
			if c == WS || isIsolateInitiator(c) || c == PDI || p.removed[j] {
				p.levels[j] = p.para
			} else {
				break
			}
		}
	}
	for i := 0; i < n; i++ {
		if c := p.orig[i]; c == B || c == S {
			p.levels[i] = p.para
			reset(i - 1)
		}
	}
	reset(n - 1)
}

// fillRemovedLevels gives each X9-removed character the embedding level of the
// nearest preceding retained character (the paragraph level if none), so the
// public level slice stays usable for reordering the full text.
func (p *paragraph) fillRemovedLevels() {
	last := p.para
	for i := range p.levels {
		if p.removed[i] {
			p.levels[i] = last
		} else {
			last = p.levels[i]
		}
	}
}

// reorderLevels implements rule L2, returning the left-to-right visual order as
// a permutation of the input indices.
func reorderLevels(levels []Level) []int {
	n := len(levels)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	highest := Level(0)
	lowestOdd := Level(1 << 30)
	for _, l := range levels {
		if l > highest {
			highest = l
		}
		if l%2 == 1 && l < lowestOdd {
			lowestOdd = l
		}
	}
	for lvl := highest; lvl >= lowestOdd; lvl-- {
		for i := 0; i < n; {
			if levels[order[i]] < lvl {
				i++
				continue
			}
			j := i
			for j < n && levels[order[j]] >= lvl {
				j++
			}
			reverse(order[i:j])
			i = j
		}
	}
	return order
}

// reverse reverses s in place.
func reverse(s []int) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// ResolveLevels runs the Unicode Bidirectional Algorithm over text and returns
// the resolved embedding level of every rune. Even levels are left-to-right,
// odd levels right-to-left. Characters removed by rule X9 (the deprecated
// embedding/override formatting characters and BN) carry the level of the
// preceding retained character.
func ResolveLevels(text []rune, base Direction) []Level {
	levels, _, _ := resolve(text, base)
	return levels
}

// BaseLevel returns the resolved paragraph embedding level chosen for text
// under base (rules P2/P3): 0 for left-to-right, 1 for right-to-left.
func BaseLevel(text []rune, base Direction) Level {
	_, _, para := resolve(text, base)
	return para
}

// Reorder implements rule L2. Given the per-rune levels (as returned by
// ResolveLevels for the same text), it returns the visual left-to-right order
// as a permutation of indices into text.
func Reorder(text []rune, levels []Level) []int {
	_ = text
	return reorderLevels(levels)
}

// VisualOrder is a convenience that resolves levels for text and returns the
// characters in visual (left-to-right) order. Characters removed by rule X9
// are dropped. Glyph mirroring (rule L4) is not applied; see the package
// documentation.
func VisualOrder(text string, base Direction) string {
	runes := []rune(text)
	levels, removed, _ := resolve(runes, base)
	order := reorderLevels(levels)
	out := make([]rune, 0, len(runes))
	for _, pos := range order {
		if removed[pos] {
			continue
		}
		out = append(out, runes[pos])
	}
	return string(out)
}
