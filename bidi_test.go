// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestConformance drives the curated subset of the Unicode
// BidiCharacterTest.txt through the public API (BaseLevel, ResolveLevels and
// Reorder) and asserts the resolved paragraph level, per-character levels and
// visual order all match the reference data.
func TestConformance(t *testing.T) {
	f, err := os.Open("testdata/BidiCharacterTest.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	cases := 0
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ";")
		if len(fields) != 5 {
			t.Fatalf("line %d: want 5 fields, got %d", lineNo, len(fields))
		}
		runes := parseRunes(fields[0])
		dir := parseDir(fields[1])
		wantPara, _ := strconv.Atoi(fields[2])
		levelToks := strings.Fields(fields[3])
		wantOrder := parseInts(fields[4])

		if got := BaseLevel(runes, dir); int(got) != wantPara {
			t.Fatalf("line %d: BaseLevel=%d want %d", lineNo, got, wantPara)
		}

		levels := ResolveLevels(runes, dir)
		for i, tok := range levelToks {
			removed := isRemovedByX9(ClassOf(runes[i]))
			if tok == "x" {
				if !removed {
					t.Fatalf("line %d: char %d expected removed", lineNo, i)
				}
				continue
			}
			if removed {
				t.Fatalf("line %d: char %d unexpectedly removed", lineNo, i)
			}
			want, _ := strconv.Atoi(tok)
			if int(levels[i]) != want {
				t.Fatalf("line %d: level[%d]=%d want %d", lineNo, i, levels[i], want)
			}
		}

		order := Reorder(runes, levels)
		var got []int
		for _, pos := range order {
			if isRemovedByX9(ClassOf(runes[pos])) {
				continue
			}
			got = append(got, pos)
		}
		if !equalInts(got, wantOrder) {
			t.Fatalf("line %d: order=%v want %v", lineNo, got, wantOrder)
		}
		cases++
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if cases == 0 {
		t.Fatal("no conformance cases parsed")
	}
	t.Logf("passed %d conformance cases", cases)
}

func parseRunes(s string) []rune {
	var out []rune
	for _, h := range strings.Fields(s) {
		v, _ := strconv.ParseUint(h, 16, 32)
		out = append(out, rune(v))
	}
	return out
}

func parseInts(s string) []int {
	var out []int
	for _, tok := range strings.Fields(s) {
		v, _ := strconv.Atoi(tok)
		out = append(out, v)
	}
	return out
}

func parseDir(s string) Direction {
	switch s {
	case "0":
		return LeftToRight
	case "1":
		return RightToLeft
	default:
		return Auto
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
