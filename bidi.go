// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

// Level is a bidirectional embedding level as defined by UAX #9. Even levels
// are left-to-right; odd levels are right-to-left.
type Level int

// maxDepth is the maximum explicit embedding depth (UAX #9 BD2).
const maxDepth = 125

// Direction selects the base paragraph direction passed to the algorithm.
type Direction int

const (
	// LeftToRight forces a base paragraph embedding level of 0.
	LeftToRight Direction = iota
	// RightToLeft forces a base paragraph embedding level of 1.
	RightToLeft
	// Auto derives the base level from the first strong character (rules P2
	// and P3), defaulting to left-to-right.
	Auto
)

// isIsolateInitiator reports whether c opens an isolate (LRI, RLI or FSI).
func isIsolateInitiator(c Class) bool { return c == LRI || c == RLI || c == FSI }

// isNI reports whether c is a neutral or isolate formatting type (the "NI" set
// used by rules N0–N2).
func isNI(c Class) bool {
	switch c {
	case B, S, WS, ON, FSI, LRI, RLI, PDI:
		return true
	}
	return false
}

// isRemovedByX9 reports whether original class c is stripped by rule X9.
func isRemovedByX9(c Class) bool {
	switch c {
	case RLE, LRE, RLO, LRO, PDF, BN:
		return true
	}
	return false
}

// embeddingDir maps a level to its strong direction (L for even, R for odd).
func embeddingDir(l Level) Class {
	if l%2 == 0 {
		return L
	}
	return R
}

// leastGreaterOdd returns the least odd level greater than l.
func leastGreaterOdd(l Level) Level {
	if l%2 == 0 {
		return l + 1
	}
	return l + 2
}

// leastGreaterEven returns the least even level greater than l.
func leastGreaterEven(l Level) Level {
	if l%2 == 0 {
		return l + 2
	}
	return l + 1
}

// strongDir maps a working type to the strong direction used by N0/N1, where
// EN and AN count as R. It returns L, R, or ON (meaning "not strong").
func strongDir(c Class) Class {
	switch c {
	case L:
		return L
	case R, EN, AN:
		return R
	}
	return ON
}

// paragraph holds the mutable state of a single UBA run over one paragraph.
type paragraph struct {
	orig    []Class // original Bidi_Class of each rune
	runes   []rune
	types   []Class // working types, mutated by W/N rules
	levels  []Level
	removed []bool // characters removed by rule X9
	para    Level  // paragraph embedding level

	matchPDI  []int // for an isolate initiator: index of matching PDI, else len(orig)
	matchInit []int // for a PDI: index of matching initiator, else -1
}

// resolve runs P2–I2 followed by L1, returning per-rune levels (removed
// characters carry the level of the preceding character) and the removed mask.
func resolve(text []rune, base Direction) (levels []Level, removed []bool, para Level) {
	p := &paragraph{runes: text}
	n := len(text)
	p.orig = make([]Class, n)
	p.types = make([]Class, n)
	p.levels = make([]Level, n)
	p.removed = make([]bool, n)
	for i, r := range text {
		c := ClassOf(r)
		p.orig[i] = c
		p.types[i] = c
	}
	p.computeIsolatePairs()
	p.para = p.baseLevel(base)
	p.explicit()
	p.resolveSequences()
	p.applyL1()
	p.fillRemovedLevels()
	return p.levels, p.removed, p.para
}
