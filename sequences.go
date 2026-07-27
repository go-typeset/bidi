// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

import "sort"

// runSequence is one isolating run sequence: the ordered original indices it
// covers, plus its start-of-sequence and end-of-sequence boundary types.
type runSequence struct {
	idx      []int
	sos, eos Class
}

// resolveSequences implements rule X10 and then applies the weak (W1–W7),
// neutral (N0–N2) and implicit (I1–I2) rules to each isolating run sequence.
func (p *paragraph) resolveSequences() {
	for _, s := range p.isolatingRunSequences() {
		p.resolveWeak(s)
		p.resolveNeutral(s)
		p.resolveImplicit(s)
	}
}

// isolatingRunSequences partitions the non-removed characters into level runs
// and chains them into isolating run sequences with their sos/eos types.
func (p *paragraph) isolatingRunSequences() []runSequence {
	// Build level runs over the characters retained after X9.
	var runs [][]int
	var cur []int
	for i, rem := range p.removed {
		if rem {
			continue
		}
		if len(cur) > 0 && p.levels[i] != p.levels[cur[len(cur)-1]] {
			runs = append(runs, cur)
			cur = nil
		}
		cur = append(cur, i)
	}
	if len(cur) > 0 {
		runs = append(runs, cur)
	}

	runOf := make(map[int]int, len(runs))
	for id, r := range runs {
		runOf[r[0]] = id
	}

	n := len(p.orig)
	var seqs []runSequence
	for id, r := range runs {
		first := r[0]
		if p.orig[first] == PDI && p.matchInit[first] >= 0 {
			continue // continuation run, consumed by its initiator's sequence
		}
		var idx []int
		cid := id
		for {
			idx = append(idx, runs[cid]...)
			last := runs[cid][len(runs[cid])-1]
			if isIsolateInitiator(p.orig[last]) && p.matchPDI[last] < n {
				// The matching PDI is retained (isolates are not removed by
				// X9) and always begins a new level run, since it returns to
				// the initiator's level after higher-level isolate content, so
				// runOf is guaranteed to contain it.
				cid = runOf[p.matchPDI[last]]
				continue
			}
			break
		}
		seqs = append(seqs, p.withBoundaries(idx))
	}
	return seqs
}

// withBoundaries computes sos and eos for the sequence covering idx (rule X10).
func (p *paragraph) withBoundaries(idx []int) runSequence {
	seqLevel := p.levels[idx[0]]

	prevLevel := p.para
	for i := idx[0] - 1; i >= 0; i-- {
		if !p.removed[i] {
			prevLevel = p.levels[i]
			break
		}
	}
	sos := embeddingDir(maxLevel(seqLevel, prevLevel))

	last := idx[len(idx)-1]
	nextLevel := p.para
	if !isIsolateInitiator(p.orig[last]) {
		for i := last + 1; i < len(p.orig); i++ {
			if !p.removed[i] {
				nextLevel = p.levels[i]
				break
			}
		}
	}
	eos := embeddingDir(maxLevel(seqLevel, nextLevel))
	return runSequence{idx: idx, sos: sos, eos: eos}
}

// maxLevel returns the larger of a and b.
func maxLevel(a, b Level) Level {
	if a > b {
		return a
	}
	return b
}

// resolveWeak applies rules W1–W7 to one isolating run sequence.
func (p *paragraph) resolveWeak(s runSequence) {
	idx := s.idx

	// W1: resolve NSM to the type of the previous character.
	prev := s.sos
	for _, ix := range idx {
		if p.types[ix] == NSM {
			if isIsolateInitiator(prev) || prev == PDI {
				p.types[ix] = ON
			} else {
				p.types[ix] = prev
			}
		}
		prev = p.types[ix]
	}

	// W2: EN becomes AN when the last strong type is AL.
	strong := s.sos
	for _, ix := range idx {
		switch c := p.types[ix]; c {
		case EN:
			if strong == AL {
				p.types[ix] = AN
			}
		case L, R, AL:
			strong = c
		}
	}

	// W3: AL becomes R.
	for _, ix := range idx {
		if p.types[ix] == AL {
			p.types[ix] = R
		}
	}

	// W4: a single ES/CS between numbers joins them.
	for k := 1; k < len(idx)-1; k++ {
		c := p.types[idx[k]]
		pr := p.types[idx[k-1]]
		nx := p.types[idx[k+1]]
		switch {
		case c == ES && pr == EN && nx == EN:
			p.types[idx[k]] = EN
		case c == CS && pr == EN && nx == EN:
			p.types[idx[k]] = EN
		case c == CS && pr == AN && nx == AN:
			p.types[idx[k]] = AN
		}
	}

	// W5: runs of ET adjacent to EN become EN.
	for k := 0; k < len(idx); {
		if p.types[idx[k]] != ET {
			k++
			continue
		}
		j := k
		for j < len(idx) && p.types[idx[j]] == ET {
			j++
		}
		adj := (k > 0 && p.types[idx[k-1]] == EN) || (j < len(idx) && p.types[idx[j]] == EN)
		if adj {
			for m := k; m < j; m++ {
				p.types[idx[m]] = EN
			}
		}
		k = j
	}

	// W6: remaining ET/ES/CS become ON.
	for _, ix := range idx {
		switch p.types[ix] {
		case ET, ES, CS:
			p.types[ix] = ON
		}
	}

	// W7: EN becomes L when the last strong type is L.
	strong = s.sos
	for _, ix := range idx {
		switch c := p.types[ix]; c {
		case EN:
			if strong == L {
				p.types[ix] = L
			}
		case L, R:
			strong = c
		}
	}
}

// resolveNeutral applies rules N0, N1 and N2 to one isolating run sequence.
func (p *paragraph) resolveNeutral(s runSequence) {
	p.resolveBrackets(s) // N0
	idx := s.idx

	// N1: NIs between matching strong directions take that direction.
	for k := 0; k < len(idx); {
		if !isNI(p.types[idx[k]]) {
			k++
			continue
		}
		j := k
		for j < len(idx) && isNI(p.types[idx[j]]) {
			j++
		}
		var before, after Class
		if k == 0 {
			before = s.sos
		} else {
			before = strongDir(p.types[idx[k-1]])
		}
		if j == len(idx) {
			after = s.eos
		} else {
			after = strongDir(p.types[idx[j]])
		}
		if before != ON && before == after {
			for m := k; m < j; m++ {
				p.types[idx[m]] = before
			}
		}
		k = j
	}

	// N2: remaining NIs take the embedding direction.
	for _, ix := range idx {
		if isNI(p.types[ix]) {
			p.types[ix] = embeddingDir(p.levels[ix])
		}
	}
}

// resolveBrackets applies rule N0 to the bracket pairs of the sequence.
func (p *paragraph) resolveBrackets(s runSequence) {
	idx := s.idx
	pairs := p.bracketPairs(idx)
	sort.Slice(pairs, func(i, j int) bool { return pairs[i][0] < pairs[j][0] })

	e := embeddingDir(p.levels[idx[0]])
	opp := R
	if e == R {
		opp = L
	}
	for _, pr := range pairs {
		o, c := pr[0], pr[1]
		foundE, foundOpp := false, false
		for m := o + 1; m < c; m++ {
			switch strongDir(p.types[idx[m]]) {
			case e:
				foundE = true
			case opp:
				foundOpp = true
			}
		}
		var dir Class
		set := false
		switch {
		case foundE:
			dir, set = e, true
		case foundOpp:
			ctx := s.sos
			for m := o - 1; m >= 0; m-- {
				if d := strongDir(p.types[idx[m]]); d != ON {
					ctx = d
					break
				}
			}
			if ctx == opp {
				dir = opp
			} else {
				dir = e
			}
			set = true
		}
		if set {
			p.types[idx[o]] = dir
			p.types[idx[c]] = dir
			p.bracketNSM(idx, o, dir)
			p.bracketNSM(idx, c, dir)
		}
	}
}

// bracketNSM applies the N0 rule that NSMs following a resolved bracket adopt
// the bracket's direction.
func (p *paragraph) bracketNSM(idx []int, pos int, dir Class) {
	for m := pos + 1; m < len(idx); m++ {
		if p.orig[idx[m]] == NSM {
			p.types[idx[m]] = dir
		} else {
			break
		}
	}
}

// resolveImplicit applies rules I1 and I2 to one isolating run sequence.
func (p *paragraph) resolveImplicit(s runSequence) {
	for _, ix := range s.idx {
		lvl := p.levels[ix]
		c := p.types[ix]
		if lvl%2 == 0 {
			switch c {
			case R:
				p.levels[ix] = lvl + 1
			case AN, EN:
				p.levels[ix] = lvl + 2
			}
		} else {
			switch c {
			case L, EN, AN:
				p.levels[ix] = lvl + 1
			}
		}
	}
}
