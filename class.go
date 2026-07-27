// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

import "sort"

// Class is a Unicode Bidi_Class property value, as defined by UAX #9 and
// UAX #44. Every rune has exactly one Bidi_Class.
type Class uint8

// The Bidi_Class values. The ordering matches the enumeration used throughout
// the Unicode Bidirectional Algorithm; do not rely on the numeric values other
// than for equality.
const (
	// Strong types.
	L  Class = iota // Left_To_Right
	R               // Right_To_Left
	AL              // Arabic_Letter

	// Weak types.
	EN  // European_Number
	ES  // European_Separator
	ET  // European_Terminator
	AN  // Arabic_Number
	CS  // Common_Separator
	NSM // Nonspacing_Mark
	BN  // Boundary_Neutral

	// Neutral types.
	B  // Paragraph_Separator
	S  // Segment_Separator
	WS // White_Space
	ON // Other_Neutral

	// Explicit formatting types.
	LRE // Left_To_Right_Embedding
	RLE // Right_To_Left_Embedding
	LRO // Left_To_Right_Override
	RLO // Right_To_Left_Override
	PDF // Pop_Directional_Format
	LRI // Left_To_Right_Isolate
	RLI // Right_To_Left_Isolate
	FSI // First_Strong_Isolate
	PDI // Pop_Directional_Isolate

	numClasses // sentinel, must stay last
)

// classNames maps each Class to its short UAX #44 abbreviation.
var classNames = [numClasses]string{
	L: "L", R: "R", AL: "AL",
	EN: "EN", ES: "ES", ET: "ET", AN: "AN", CS: "CS", NSM: "NSM", BN: "BN",
	B: "B", S: "S", WS: "WS", ON: "ON",
	LRE: "LRE", RLE: "RLE", LRO: "LRO", RLO: "RLO", PDF: "PDF",
	LRI: "LRI", RLI: "RLI", FSI: "FSI", PDI: "PDI",
}

// String returns the short Bidi_Class abbreviation (for example "AL"). An
// out-of-range Class value renders as "Class(N)".
func (c Class) String() string {
	if int(c) >= len(classNames) {
		return "Class(" + itoa(int(c)) + ")"
	}
	return classNames[c]
}

// itoa formats a small non-negative int without importing strconv, keeping the
// runtime dependency surface to sort only.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// ClassOf returns the Bidi_Class of r. Runes outside the assigned ranges
// follow the Unicode defaults declared by the @missing lines of
// DerivedBidiClass.txt (L in general, R/AL in unassigned right-to-left blocks,
// ET in the currency block, BN for default-ignorable and non-characters).
//
// It is named ClassOf rather than Class because Go does not allow a function
// to share the name of the Class type it returns.
func ClassOf(r rune) Class {
	// bidiRanges is generated, sorted by Lo, and covers all of 0..0x10FFFF, so
	// the search below always resolves to a range.
	i := sort.Search(len(bidiRanges), func(i int) bool {
		return bidiRanges[i].Hi >= uint32(r)
	})
	if i < len(bidiRanges) && uint32(r) >= bidiRanges[i].Lo {
		return bidiRanges[i].Class
	}
	return L
}

// bidiRange is one contiguous run of code points sharing a Bidi_Class.
type bidiRange struct {
	Lo, Hi uint32
	Class  Class
}
