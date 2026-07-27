// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package bidi is a pure-Go, CGO=0, standard-library-only implementation of the
// Unicode Bidirectional Algorithm (UAX #9) for laying out mixed
// left-to-right / right-to-left text.
//
// It has no dependency on golang.org/x/text: the Bidi_Class property and the
// paired-bracket property are shipped as generated lookup tables (see
// cmd/genbidi), so the package builds anywhere the standard library does.
//
// # Public API
//
//   - [ClassOf] reports the Bidi_Class of a rune, and the [Class] enum
//     enumerates every Bidi_Class value.
//   - [ResolveLevels] runs the algorithm over a paragraph and returns the
//     resolved embedding [Level] of each rune.
//   - [BaseLevel] reports the paragraph embedding level chosen by rules P2/P3.
//   - [Reorder] applies rule L2 to a level slice, returning the visual-order
//     index permutation; [ReorderWithMarks] additionally applies the rule L3
//     combining-mark refinement.
//   - [VisualOrder] is a convenience that resolves and reorders a string in one
//     call.
//   - [Paragraphs] splits text into paragraphs (rule P1), and
//     [VisualParagraphs] runs the whole display pipeline per paragraph.
//   - [Mirror] and [MirrorRunes] apply glyph mirroring (rule L4).
//   - [JoinForms] resolves Arabic cursive presentation forms, and
//     [PresentationForm] maps a letter to the Arabic Presentation Forms-B block.
//   - [Direction] selects the base direction: [LeftToRight], [RightToLeft] or
//     [Auto].
//
// # Implemented rules
//
// The core algorithm runs through rule L2, the full extent verified by the
// Unicode conformance file BidiCharacterTest.txt, with rules L3, L4 and P1 and
// Arabic joining layered on top for display:
//
//   - P1: splitting text into paragraphs on Paragraph_Separator ([Paragraphs]).
//   - P2, P3: paragraph embedding level from the first strong character.
//   - X1–X8: explicit embeddings (LRE/RLE/LRO/RLO/PDF) and isolates
//     (LRI/RLI/FSI/PDI), including overflow handling and the directional
//     status stack, with FSI resolved per X5c.
//   - X9: the deprecated embedding/override/PDF formatting characters and BN
//     are removed (reported as carrying the preceding character's level).
//   - X10: isolating run sequences with their sos/eos boundary types.
//   - W1–W7: the weak-type rules.
//   - N0: paired-bracket resolution (BD16), including the U+2329/U+232A ~
//     U+3008/U+3009 canonical equivalence.
//   - N1, N2: the neutral-type rules.
//   - I1, I2: the implicit level rules.
//   - L1: resetting separators and trailing whitespace to the paragraph level.
//   - L2: reordering to visual order.
//   - L3: keeping combining marks adjacent to their base after reordering
//     ([ReorderWithMarks]); the plain [Reorder] applies L2 only, matching the
//     conformance data.
//   - L4: substituting mirrored glyphs for characters at right-to-left levels
//     ([Mirror], [MirrorRunes]).
//
// It also provides the Unicode-level Arabic cursive joining algorithm
// ([JoinForms]) and a static fallback to the Arabic Presentation Forms-B block
// ([PresentationForm]).
//
// The package is validated against the full BidiCharacterTest.txt (all cases
// pass); a curated subset is embedded under testdata for the committed test
// suite.
//
// # Deferred
//
//   - Full contextual shaping: [JoinForms] resolves the isolated/initial/
//     medial/final form of each Arabic letter, but real display requires the
//     font's GSUB init/medi/fina/isol features and ligatures (such as the
//     mandatory LAM+ALEF ligature). [PresentationForm] is only a static
//     per-letter fallback for the common letters, not a shaper.
package bidi
