// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

// bracketKind classifies a paired-bracket character (property
// Bidi_Paired_Bracket_Type).
type bracketKind uint8

const (
	bracketNone bracketKind = iota
	bracketOpen
	bracketClose
)

// bracketInfo is one entry of the generated bracket table: the kind of bracket
// and the code point of its partner (Bidi_Paired_Bracket).
type bracketInfo struct {
	kind bracketKind
	pair rune
}

// canonBracket applies the two canonical-equivalence identities relevant to
// bracket pairing (BD16): U+2329/U+232A are treated as U+3008/U+3009.
func canonBracket(r rune) rune {
	switch r {
	case 0x2329:
		return 0x3008
	case 0x232A:
		return 0x3009
	}
	return r
}

// bracketStackLimit is the fixed BD16 stack capacity.
const bracketStackLimit = 63

// bracketPairs implements BD16: it returns the bracket pairs found in the
// sequence covering idx as [openK, closeK] positions within idx.
func (p *paragraph) bracketPairs(idx []int) [][2]int {
	type entry struct {
		close rune
		k     int
	}
	var stack []entry
	var pairs [][2]int
	for k, ix := range idx {
		if p.types[ix] != ON {
			continue
		}
		info, ok := bracketData[p.runes[ix]]
		if !ok {
			continue
		}
		switch info.kind {
		case bracketOpen:
			if len(stack) >= bracketStackLimit {
				return pairs // stack full: stop processing BD16
			}
			stack = append(stack, entry{canonBracket(info.pair), k})
		case bracketClose:
			cc := canonBracket(p.runes[ix])
			for si := len(stack) - 1; si >= 0; si-- {
				if stack[si].close == cc {
					pairs = append(pairs, [2]int{stack[si].k, k})
					stack = stack[:si]
					break
				}
			}
		}
	}
	return pairs
}
