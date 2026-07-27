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
//     index permutation.
//   - [VisualOrder] is a convenience that resolves and reorders a string in one
//     call.
//   - [Direction] selects the base direction: [LeftToRight], [RightToLeft] or
//     [Auto].
//
// # Implemented rules
//
// The algorithm is implemented through rule L2, which is the full extent
// verified by the Unicode conformance file BidiCharacterTest.txt:
//
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
//
// The package is validated against the full BidiCharacterTest.txt (all cases
// pass); a curated subset is embedded under testdata for the committed test
// suite.
//
// # Deferred
//
//   - L3 (combining marks) and L4 (mirroring of paired-bracket and other
//     mirrored glyphs) are out of scope: they belong to the rendering/shaping
//     stage. [VisualOrder] therefore does not substitute mirrored glyphs.
//   - Arabic cursive shaping and joining are the job of a shaper, not the
//     bidirectional algorithm, and are out of scope here.
//   - Rule P1 (splitting text into paragraphs on Paragraph_Separator) is the
//     caller's responsibility; the API operates on a single paragraph, though a
//     Paragraph_Separator encountered inline is handled by rule X8/L1.
package bidi
