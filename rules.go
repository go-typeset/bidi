// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

// computeIsolatePairs fills matchPDI and matchInit by scanning for isolate
// initiators (LRI/RLI/FSI) and their matching PDI (rule BD9).
func (p *paragraph) computeIsolatePairs() {
	n := len(p.orig)
	p.matchPDI = make([]int, n)
	p.matchInit = make([]int, n)
	for i := range p.matchPDI {
		p.matchPDI[i] = n
		p.matchInit[i] = -1
	}
	var stack []int
	for i, c := range p.orig {
		switch {
		case isIsolateInitiator(c):
			stack = append(stack, i)
		case c == PDI:
			if len(stack) > 0 {
				j := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				p.matchPDI[j] = i
				p.matchInit[i] = j
			}
		}
	}
}

// baseLevel implements P2/P3, honouring the requested Direction.
func (p *paragraph) baseLevel(base Direction) Level {
	switch base {
	case LeftToRight:
		return 0
	case RightToLeft:
		return 1
	}
	// Auto: first strong character in the paragraph, skipping isolates.
	if d := p.firstStrong(0, len(p.orig)); d == R {
		return 1
	}
	return 0
}

// firstStrong returns the first strong direction (L or R) in [start,end),
// skipping the contents of nested isolates (rules P2/P3 and X5c). AL counts as
// R. It returns ON when no strong character is present.
func (p *paragraph) firstStrong(start, end int) Class {
	depth := 0
	for i := start; i < end; i++ {
		c := p.orig[i]
		switch {
		case isIsolateInitiator(c):
			depth++
		case c == PDI:
			if depth > 0 {
				depth--
			}
		case depth == 0:
			switch c {
			case L:
				return L
			case R, AL:
				return R
			}
		}
	}
	return ON
}

// statusEntry is one entry on the directional status stack (rule X1).
type statusEntry struct {
	level    Level
	override Class // L, R, or ON (neutral)
	isolate  bool
}

// explicit implements rules X1–X8: it assigns each character an embedding
// level, applies directional overrides to the working types, resolves FSI to
// LRI/RLI, and marks the characters removed by X9.
func (p *paragraph) explicit() {
	stack := []statusEntry{{p.para, ON, false}}
	top := func() statusEntry { return stack[len(stack)-1] }

	overflowIsolate := 0
	overflowEmbedding := 0
	validIsolate := 0

	for i, c := range p.orig {
		switch c {
		case RLE, LRE, RLO, LRO:
			p.levels[i] = top().level
			p.removed[i] = true
			var newLevel Level
			if c == RLE || c == RLO {
				newLevel = leastGreaterOdd(top().level)
			} else {
				newLevel = leastGreaterEven(top().level)
			}
			if newLevel <= maxDepth && overflowIsolate == 0 && overflowEmbedding == 0 {
				ov := ON
				switch c {
				case LRO:
					ov = L
				case RLO:
					ov = R
				}
				stack = append(stack, statusEntry{newLevel, ov, false})
			} else if overflowIsolate == 0 {
				overflowEmbedding++
			}

		case RLI, LRI, FSI:
			dir := c
			if c == FSI {
				if p.firstStrong(i+1, p.matchPDI[i]) == R {
					dir = RLI
				} else {
					dir = LRI
				}
			}
			p.levels[i] = top().level
			if top().override != ON {
				p.types[i] = top().override
			}
			var newLevel Level
			if dir == RLI {
				newLevel = leastGreaterOdd(top().level)
			} else {
				newLevel = leastGreaterEven(top().level)
			}
			if newLevel <= maxDepth && overflowIsolate == 0 && overflowEmbedding == 0 {
				validIsolate++
				stack = append(stack, statusEntry{newLevel, ON, true})
			} else {
				overflowIsolate++
			}

		case PDI:
			if overflowIsolate > 0 {
				overflowIsolate--
			} else if validIsolate > 0 {
				overflowEmbedding = 0
				for !top().isolate {
					stack = stack[:len(stack)-1]
				}
				stack = stack[:len(stack)-1]
				validIsolate--
			}
			p.levels[i] = top().level
			if top().override != ON {
				p.types[i] = top().override
			}

		case PDF:
			p.removed[i] = true
			if overflowIsolate > 0 {
				// ignore
			} else if overflowEmbedding > 0 {
				overflowEmbedding--
			} else if !top().isolate && len(stack) >= 2 {
				stack = stack[:len(stack)-1]
			}
			p.levels[i] = top().level

		case B:
			// Rule X8: a paragraph separator resets to the paragraph level.
			stack = stack[:1]
			overflowIsolate = 0
			overflowEmbedding = 0
			validIsolate = 0
			p.levels[i] = p.para

		case BN:
			p.levels[i] = top().level
			p.removed[i] = true

		default:
			p.levels[i] = top().level
			if top().override != ON {
				p.types[i] = top().override
			}
		}
	}
}
