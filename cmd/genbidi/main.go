// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Command genbidi fetches the Unicode Character Database DerivedBidiClass.txt
// and BidiBrackets.txt files and emits the compact Go lookup tables used by
// package bidi: bidiclass_table.go (a sorted Bidi_Class range table) and
// bidibrackets_table.go (the paired-bracket table for rule N0).
//
// Usage:
//
//	go run ./cmd/genbidi          # writes the tables into the current directory
//	go run ./cmd/genbidi outdir   # writes the tables into outdir
//
// The generated tables are committed to the repository; genbidi only needs to
// be re-run when tracking a newer Unicode version.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// sourceURL is the canonical location of the derived Bidi_Class data.
const sourceURL = "https://www.unicode.org/Public/UCD/latest/ucd/extracted/DerivedBidiClass.txt"

// bracketURL is the canonical location of the paired-bracket data.
const bracketURL = "https://www.unicode.org/Public/UCD/latest/ucd/BidiBrackets.txt"

// maxRune is the highest Unicode code point, inclusive.
const maxRune = 0x10FFFF

// Seams so tests can drive genbidi hermetically without real network or exits.
var (
	httpGet           = http.Get
	osExit            = os.Exit
	stderr  io.Writer = os.Stderr
	args              = os.Args
)

func main() { osExit(run(args)) }

// run performs the fetch/parse/generate/write pipeline for both generated
// tables and returns a process exit code (0 on success, 1 on any failure). An
// optional first argument selects the output directory (default ".").
func run(argv []string) int {
	dir := "."
	if len(argv) > 1 {
		dir = argv[1]
	}
	// Bidi_Class range table.
	data, err := fetch(sourceURL)
	if err != nil {
		fmt.Fprintln(stderr, "genbidi: fetch:", err)
		return 1
	}
	ranges, err := parse(data)
	if err != nil {
		fmt.Fprintln(stderr, "genbidi: parse:", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(dir, "bidiclass_table.go"), generate(ranges), 0o644); err != nil {
		fmt.Fprintln(stderr, "genbidi: write:", err)
		return 1
	}
	// Paired-bracket table.
	bdata, err := fetch(bracketURL)
	if err != nil {
		fmt.Fprintln(stderr, "genbidi: fetch brackets:", err)
		return 1
	}
	brackets, err := parseBrackets(bdata)
	if err != nil {
		fmt.Fprintln(stderr, "genbidi: parse brackets:", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(dir, "bidibrackets_table.go"), generateBrackets(brackets), 0o644); err != nil {
		fmt.Fprintln(stderr, "genbidi: write brackets:", err)
		return 1
	}
	fmt.Fprintf(stderr, "genbidi: wrote %d ranges and %d brackets\n", len(ranges), len(brackets))
	return 0
}

// fetch retrieves url and returns its body, failing on transport errors and on
// any non-200 status.
func fetch(url string) ([]byte, error) {
	resp, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// genRange is one contiguous run of code points sharing a Bidi_Class, named by
// its UAX #44 abbreviation (for example "AL").
type genRange struct {
	lo, hi uint32
	class  string
}

// abbrevs is the ordered list of Bidi_Class abbreviations; index into it is the
// per-code-point value used while building the table.
var abbrevs = []string{
	"L", "R", "AL", "EN", "ES", "ET", "AN", "CS", "NSM", "BN",
	"B", "S", "WS", "ON", "LRE", "RLE", "LRO", "RLO", "PDF",
	"LRI", "RLI", "FSI", "PDI",
}

// fullNames maps the long Bidi_Class names used on @missing lines to their
// abbreviation.
var fullNames = map[string]string{
	"Left_To_Right": "L", "Right_To_Left": "R", "Arabic_Letter": "AL",
	"European_Number": "EN", "European_Separator": "ES", "European_Terminator": "ET",
	"Arabic_Number": "AN", "Common_Separator": "CS", "Nonspacing_Mark": "NSM",
	"Boundary_Neutral": "BN", "Paragraph_Separator": "B", "Segment_Separator": "S",
	"White_Space": "WS", "Other_Neutral": "ON", "Left_To_Right_Embedding": "LRE",
	"Right_To_Left_Embedding": "RLE", "Left_To_Right_Override": "LRO",
	"Right_To_Left_Override": "RLO", "Pop_Directional_Format": "PDF",
	"Left_To_Right_Isolate": "LRI", "Right_To_Left_Isolate": "RLI",
	"First_Strong_Isolate": "FSI", "Pop_Directional_Isolate": "PDI",
}

// classIndex maps an abbreviation to its position in abbrevs.
var classIndex = func() map[string]uint8 {
	m := make(map[string]uint8, len(abbrevs))
	for i, a := range abbrevs {
		m[a] = uint8(i)
	}
	return m
}()

// resolveClass turns a token from either a data line (abbreviation) or an
// @missing line (long name) into an abbreviation, reporting unknown tokens.
func resolveClass(tok string) (string, error) {
	if _, ok := classIndex[tok]; ok {
		return tok, nil
	}
	if a, ok := fullNames[tok]; ok {
		return a, nil
	}
	return "", fmt.Errorf("unknown Bidi_Class %q", tok)
}

// parse builds the full code-point→Bidi_Class mapping by applying, in file
// order, the @missing defaults and the explicit data lines, then run-length
// encodes the result into a sorted slice of ranges covering 0..maxRune.
func parse(data []byte) ([]genRange, error) {
	table := make([]uint8, maxRune+1) // 0 == "L", the global default

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(raw, "\r")
		if m, ok := missingBody(line); ok {
			lo, hi, class, err := parseAssignment(m)
			if err != nil {
				return nil, err
			}
			fill(table, lo, hi, class)
			continue
		}
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		lo, hi, class, err := parseAssignment(line)
		if err != nil {
			return nil, err
		}
		fill(table, lo, hi, class)
	}
	return encode(table), nil
}

// missingBody reports whether line is an "# @missing:" directive and, if so,
// returns the assignment text that follows the marker.
func missingBody(line string) (string, bool) {
	const marker = "# @missing:"
	if strings.HasPrefix(line, marker) {
		return strings.TrimSpace(line[len(marker):]), true
	}
	return "", false
}

// parseAssignment parses a "LO[..HI]; Class" assignment, returning the class as
// an abbreviation-index into abbrevs.
func parseAssignment(s string) (lo, hi uint32, class uint8, err error) {
	semi := strings.IndexByte(s, ';')
	if semi < 0 {
		return 0, 0, 0, fmt.Errorf("missing ';' in %q", s)
	}
	rng := strings.TrimSpace(s[:semi])
	tok := strings.TrimSpace(s[semi+1:])
	abbr, err := resolveClass(tok)
	if err != nil {
		return 0, 0, 0, err
	}
	if dots := strings.Index(rng, ".."); dots >= 0 {
		if lo, err = parseHex(rng[:dots]); err != nil {
			return 0, 0, 0, err
		}
		if hi, err = parseHex(rng[dots+2:]); err != nil {
			return 0, 0, 0, err
		}
	} else {
		if lo, err = parseHex(rng); err != nil {
			return 0, 0, 0, err
		}
		hi = lo
	}
	return lo, hi, classIndex[abbr], nil
}

// parseHex parses a hexadecimal code point, rejecting values above maxRune.
func parseHex(s string) (uint32, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 16, 32)
	if err != nil {
		return 0, err
	}
	if v > maxRune {
		return 0, fmt.Errorf("code point %X out of range", v)
	}
	return uint32(v), nil
}

// fill sets table[lo..hi] to class.
func fill(table []uint8, lo, hi uint32, class uint8) {
	for i := lo; i <= hi; i++ {
		table[i] = class
	}
}

// encode run-length encodes the per-code-point table into contiguous ranges.
func encode(table []uint8) []genRange {
	var out []genRange
	start := uint32(0)
	cur := table[0]
	for i := uint32(1); i <= maxRune; i++ {
		if table[i] != cur {
			out = append(out, genRange{start, i - 1, abbrevs[cur]})
			start = i
			cur = table[i]
		}
	}
	out = append(out, genRange{start, maxRune, abbrevs[cur]})
	return out
}

// bracketRow is one paired-bracket entry from BidiBrackets.txt.
type bracketRow struct {
	cp   uint32
	pair uint32
	kind string // "bracketOpen" or "bracketClose"
}

// parseBrackets parses BidiBrackets.txt into its bracket rows.
func parseBrackets(data []byte) ([]bracketRow, error) {
	var rows []bracketRow
	for _, raw := range strings.Split(string(data), "\n") {
		line := raw
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, ";")
		if len(f) != 3 {
			return nil, fmt.Errorf("malformed bracket line %q", raw)
		}
		cp, err := parseHex(f[0])
		if err != nil {
			return nil, err
		}
		pair, err := parseHex(f[1])
		if err != nil {
			return nil, err
		}
		var kind string
		switch strings.TrimSpace(f[2]) {
		case "o":
			kind = "bracketOpen"
		case "c":
			kind = "bracketClose"
		default:
			return nil, fmt.Errorf("unknown bracket type %q", f[2])
		}
		rows = append(rows, bracketRow{cp, pair, kind})
	}
	return rows, nil
}

// generateBrackets renders rows as the bidibrackets_table.go source file.
func generateBrackets(rows []bracketRow) []byte {
	var b strings.Builder
	b.WriteString("// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.\n")
	b.WriteString("// Use of this source code is governed by a BSD-3-Clause license that can be\n")
	b.WriteString("// found in the LICENSE file at the root of this repository.\n\n")
	b.WriteString("// Code generated by cmd/genbidi from the Unicode Character Database\n")
	b.WriteString("// BidiBrackets.txt. DO NOT EDIT.\n\n")
	b.WriteString("package bidi\n\n")
	b.WriteString("// bracketData maps each paired-bracket character to its kind and partner,\n")
	b.WriteString("// as used by rule N0 (BD16).\n")
	b.WriteString("var bracketData = map[rune]bracketInfo{\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "\t0x%04X: {%s, 0x%04X},\n", r.cp, r.kind, r.pair)
	}
	b.WriteString("}\n")
	return []byte(b.String())
}

// generate renders ranges as the bidiclass_table.go source file.
func generate(ranges []genRange) []byte {
	var b strings.Builder
	b.WriteString("// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.\n")
	b.WriteString("// Use of this source code is governed by a BSD-3-Clause license that can be\n")
	b.WriteString("// found in the LICENSE file at the root of this repository.\n\n")
	b.WriteString("// Code generated by cmd/genbidi from the Unicode Character Database\n")
	b.WriteString("// DerivedBidiClass.txt. DO NOT EDIT.\n\n")
	b.WriteString("package bidi\n\n")
	fmt.Fprintf(&b, "// bidiRanges holds the Bidi_Class of every code point as %d sorted,\n", len(ranges))
	b.WriteString("// non-overlapping ranges covering all of 0..0x10FFFF.\n")
	b.WriteString("var bidiRanges = []bidiRange{\n")
	for _, r := range ranges {
		fmt.Fprintf(&b, "\t{0x%04X, 0x%04X, %s},\n", r.lo, r.hi, r.class)
	}
	b.WriteString("}\n")
	return []byte(b.String())
}
