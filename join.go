// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

// JoiningType is the Arabic Joining_Type property of a character (UAX #9 /
// Section 9.2 of the Unicode Standard), which drives cursive joining.
type JoiningType uint8

const (
	// joinU is Non_Joining, the default for characters absent from the table.
	joinU JoiningType = iota
	// joinR is Right_Joining: joins only with a preceding letter.
	joinR
	// joinL is Left_Joining: joins only with a following letter.
	joinL
	// joinD is Dual_Joining: joins on both sides.
	joinD
	// joinC is Join_Causing (for example TATWEEL and ZERO WIDTH JOINER): joins
	// on both sides and forces adjacent letters to join.
	joinC
	// joinT is Transparent (combining marks and similar): skipped when
	// determining the joining of surrounding letters.
	joinT
)

// joiningTypeOf returns the Arabic joining type of r. Characters listed in the
// generated table use their listed type. As specified by ArabicShaping.txt,
// unlisted combining marks (Bidi_Class NSM, i.e. General_Category Mn/Me) default
// to Transparent so they do not break the cursive joining of surrounding
// letters; all other unlisted characters default to Non_Joining.
func joiningTypeOf(r rune) JoiningType {
	if t, ok := joiningData[r]; ok {
		return t
	}
	if ClassOf(r) == NSM {
		return joinT
	}
	return joinU
}

// linkPrev reports whether a letter of joining type t can connect to the letter
// preceding it in logical order (Right_Joining, Dual_Joining or Join_Causing).
func linkPrev(t JoiningType) bool {
	return t == joinR || t == joinD || t == joinC
}

// linkNext reports whether a letter of joining type t can connect to the letter
// following it in logical order (Left_Joining, Dual_Joining or Join_Causing).
func linkNext(t JoiningType) bool {
	return t == joinL || t == joinD || t == joinC
}

// JoinForm is the cursive presentation form an Arabic-script letter takes,
// according to its neighbours.
type JoinForm int

const (
	// Isolated is the standalone form (no cursive connection).
	Isolated JoinForm = iota
	// Initial connects only to the following letter.
	Initial
	// Medial connects to both neighbours.
	Medial
	// Final connects only to the preceding letter.
	Final
)

// JoinForms resolves the cursive presentation form of every character in text
// per the Arabic joining algorithm. Transparent characters (combining marks)
// are reported as [Isolated] and are skipped when joining their neighbours; a
// letter joins to an adjacent letter only when both are able to connect on the
// shared side.
//
// The result is the Unicode-level joining form; a full shaper additionally
// applies the font's GSUB init/medi/fina/isol features and ligatures (such as
// the mandatory LAM+ALEF ligature). See [PresentationForm] for a static
// fallback mapping to the Arabic Presentation Forms-B block.
func JoinForms(text []rune) []JoinForm {
	n := len(text)
	types := make([]JoiningType, n)
	for i, r := range text {
		types[i] = joiningTypeOf(r)
	}
	out := make([]JoinForm, n)
	for i := 0; i < n; i++ {
		if types[i] == joinT {
			out[i] = Isolated
			continue
		}
		prev := joinU
		for j := i - 1; j >= 0; j-- {
			if types[j] != joinT {
				prev = types[j]
				break
			}
		}
		next := joinU
		for j := i + 1; j < n; j++ {
			if types[j] != joinT {
				next = types[j]
				break
			}
		}
		joinsPrev := linkPrev(types[i]) && linkNext(prev)
		joinsNext := linkNext(types[i]) && linkPrev(next)
		switch {
		case joinsPrev && joinsNext:
			out[i] = Medial
		case joinsPrev:
			out[i] = Final
		case joinsNext:
			out[i] = Initial
		default:
			out[i] = Isolated
		}
	}
	return out
}

// presFormsB maps a base Arabic letter to its glyphs in the Arabic Presentation
// Forms-B block, indexed by [JoinForm] ([Isolated], [Initial], [Medial],
// [Final]). A zero entry means the letter has no glyph for that form (for
// example a right-joining letter has no initial or medial form).
var presFormsB = map[rune][4]rune{
	0x0621: {0xFE80, 0, 0, 0},                // HAMZA
	0x0622: {0xFE81, 0, 0, 0xFE82},           // ALEF WITH MADDA ABOVE
	0x0623: {0xFE83, 0, 0, 0xFE84},           // ALEF WITH HAMZA ABOVE
	0x0624: {0xFE85, 0, 0, 0xFE86},           // WAW WITH HAMZA ABOVE
	0x0625: {0xFE87, 0, 0, 0xFE88},           // ALEF WITH HAMZA BELOW
	0x0626: {0xFE89, 0xFE8B, 0xFE8C, 0xFE8A}, // YEH WITH HAMZA ABOVE
	0x0627: {0xFE8D, 0, 0, 0xFE8E},           // ALEF
	0x0628: {0xFE8F, 0xFE91, 0xFE92, 0xFE90}, // BEH
	0x0629: {0xFE93, 0, 0, 0xFE94},           // TEH MARBUTA
	0x062A: {0xFE95, 0xFE97, 0xFE98, 0xFE96}, // TEH
	0x062B: {0xFE99, 0xFE9B, 0xFE9C, 0xFE9A}, // THEH
	0x062C: {0xFE9D, 0xFE9F, 0xFEA0, 0xFE9E}, // JEEM
	0x062D: {0xFEA1, 0xFEA3, 0xFEA4, 0xFEA2}, // HAH
	0x062E: {0xFEA5, 0xFEA7, 0xFEA8, 0xFEA6}, // KHAH
	0x062F: {0xFEA9, 0, 0, 0xFEAA},           // DAL
	0x0630: {0xFEAB, 0, 0, 0xFEAC},           // THAL
	0x0631: {0xFEAD, 0, 0, 0xFEAE},           // REH
	0x0632: {0xFEAF, 0, 0, 0xFEB0},           // ZAIN
	0x0633: {0xFEB1, 0xFEB3, 0xFEB4, 0xFEB2}, // SEEN
	0x0634: {0xFEB5, 0xFEB7, 0xFEB8, 0xFEB6}, // SHEEN
	0x0635: {0xFEB9, 0xFEBB, 0xFEBC, 0xFEBA}, // SAD
	0x0636: {0xFEBD, 0xFEBF, 0xFEC0, 0xFEBE}, // DAD
	0x0637: {0xFEC1, 0xFEC3, 0xFEC4, 0xFEC2}, // TAH
	0x0638: {0xFEC5, 0xFEC7, 0xFEC8, 0xFEC6}, // ZAH
	0x0639: {0xFEC9, 0xFECB, 0xFECC, 0xFECA}, // AIN
	0x063A: {0xFECD, 0xFECF, 0xFED0, 0xFECE}, // GHAIN
	0x0641: {0xFED1, 0xFED3, 0xFED4, 0xFED2}, // FEH
	0x0642: {0xFED5, 0xFED7, 0xFED8, 0xFED6}, // QAF
	0x0643: {0xFED9, 0xFEDB, 0xFEDC, 0xFEDA}, // KAF
	0x0644: {0xFEDD, 0xFEDF, 0xFEE0, 0xFEDE}, // LAM
	0x0645: {0xFEE1, 0xFEE3, 0xFEE4, 0xFEE2}, // MEEM
	0x0646: {0xFEE5, 0xFEE7, 0xFEE8, 0xFEE6}, // NOON
	0x0647: {0xFEE9, 0xFEEB, 0xFEEC, 0xFEEA}, // HEH
	0x0648: {0xFEED, 0, 0, 0xFEEE},           // WAW
	0x0649: {0xFEEF, 0, 0, 0xFEF0},           // ALEF MAKSURA
	0x064A: {0xFEF1, 0xFEF3, 0xFEF4, 0xFEF2}, // YEH
}

// PresentationForm returns the Arabic Presentation Forms-B code point for the
// letter r in cursive form (as produced by [JoinForms]). It is a static
// fallback for common letters only: when r has no mapping, or no glyph for the
// requested form, r is returned unchanged.
//
// This is the Unicode-level fallback. Correct rendering requires the font's
// GSUB init/medi/fina/isol features and contextual ligatures, which this
// package does not perform.
func PresentationForm(r rune, form JoinForm) rune {
	if forms, ok := presFormsB[r]; ok && uint(form) < uint(len(forms)) {
		if g := forms[form]; g != 0 {
			return g
		}
	}
	return r
}
