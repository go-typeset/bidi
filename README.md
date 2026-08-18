# bidi

[![CI](https://github.com/go-typeset/bidi/actions/workflows/ci.yml/badge.svg)](https://github.com/go-typeset/bidi/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-typeset/bidi.svg)](https://pkg.go.dev/github.com/go-typeset/bidi)
![coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)
![go](https://img.shields.io/badge/Go-1.26.4%2B-00ADD8?logo=go&logoColor=white)
[![License: BSD-3-Clause](https://img.shields.io/badge/License-BSD--3--Clause-blue.svg)](LICENSE)

A pure-Go, `CGO_ENABLED=0`, **standard-library-only** implementation of the
[Unicode Bidirectional Algorithm (UAX #9)](https://www.unicode.org/reports/tr9/)
for laying out mixed left-to-right / right-to-left text.

Unlike most Go bidi implementations, `go-opentype/bidi` does **not** depend on
`golang.org/x/text`. The `Bidi_Class` and paired-bracket Unicode properties are
compiled into small generated lookup tables (see [`cmd/genbidi`](cmd/genbidi)),
so the package builds anywhere the standard library does.

## Install

```sh
go get github.com/go-typeset/bidi
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-typeset/bidi"
)

func main() {
	// A logical-order string mixing English and Hebrew.
	s := "abc אבג def"

	// Visual (left-to-right) order for display.
	fmt.Println(bidi.VisualOrder(s, bidi.Auto))

	// Or work with levels directly.
	runes := []rune(s)
	levels := bidi.ResolveLevels(runes, bidi.LeftToRight)
	order := bidi.Reorder(runes, levels) // visual-order permutation of indices
	fmt.Println(levels, order)

	// Inspect a single rune's Bidi_Class.
	fmt.Println(bidi.ClassOf('א')) // R
}
```

See [`example_test.go`](./example_test.go) for runnable examples of each of
the functions above, and `go doc github.com/go-typeset/bidi` for the full
reference.

## API

| Symbol | Purpose |
| --- | --- |
| `ClassOf(r rune) Class` | Bidi_Class of a rune |
| `Class` enum | `L R AL EN ES ET AN CS NSM BN B S WS ON LRE RLE LRO RLO PDF LRI RLI FSI PDI` |
| `ResolveLevels(text []rune, base Direction) []Level` | resolved embedding level per rune |
| `BaseLevel(text []rune, base Direction) Level` | paragraph level (rules P2/P3) |
| `Reorder(text []rune, levels []Level) []int` | rule L2 visual-order permutation |
| `ReorderWithMarks(text []rune, levels []Level) []int` | L2 + rule L3 combining-mark reorder |
| `VisualOrder(text string, base Direction) string` | resolve + reorder convenience |
| `Paragraphs(text []rune) [][]rune` | rule P1 paragraph splitting |
| `VisualParagraphs(text string, base Direction) []string` | full display pipeline per paragraph |
| `Mirror(r rune) rune` | rule L4 mirrored glyph of a rune |
| `MirrorRunes(text []rune, levels []Level) []rune` | mirror characters at RTL levels |
| `JoinForms(text []rune) []JoinForm` | Arabic cursive form per character |
| `PresentationForm(r rune, form JoinForm) rune` | Arabic Presentation Forms-B fallback |
| `JoinForm` | `Isolated`, `Initial`, `Medial`, `Final` |
| `Direction` | `LeftToRight`, `RightToLeft`, `Auto` |
| `Level` | embedding level (even = LTR, odd = RTL) |

## Implemented vs deferred

**Implemented** — the core algorithm runs through rule **L2**, the full extent
covered by the Unicode conformance file `BidiCharacterTest.txt`, with **P1**,
**L3**, **L4** and Arabic joining layered on top for display:

- **P1** paragraph splitting on `Paragraph_Separator` (`Paragraphs`), with
  CR+LF treated as a single separator.
- **P2, P3** base paragraph level from the first strong character.
- **X1–X8** explicit embeddings *and* isolates (with overflow handling and the
  directional status stack); **X9** removal of the deprecated formatting
  characters and `BN`; **X10** isolating run sequences with `sos`/`eos`.
- **W1–W7** weak types.
- **N0** paired-bracket resolution (BD16, incl. the U+2329/U+232A canonical
  equivalence), **N1–N2** neutral types.
- **I1, I2** implicit levels.
- **L1** separator / trailing-whitespace reset, **L2** reordering.
- **L3** combining marks kept adjacent to their base after reordering
  (`ReorderWithMarks`; plain `Reorder` stays L2-only to match the conformance
  data).
- **L4** glyph mirroring for characters at right-to-left levels (`Mirror`,
  `MirrorRunes`).
- Arabic cursive **joining** at the Unicode level (`JoinForms`), plus a static
  Arabic Presentation Forms-B fallback (`PresentationForm`).

**Deferred**:

- Full contextual **shaping** — `JoinForms` resolves each letter's
  isolated/initial/medial/final form, but real rendering needs the font's GSUB
  `init`/`medi`/`fina`/`isol` features and contextual ligatures (such as the
  mandatory LAM+ALEF ligature). `PresentationForm` is only a per-letter fallback
  for the common letters, not a shaper.

## Conformance

The package is validated against the **entire** `BidiCharacterTest.txt`
(all cases pass: paragraph level, per-character levels and visual order). A
curated representative subset is embedded under [`testdata`](testdata) and run by
`TestConformance`. CI enforces **exactly 100%** statement coverage, `go vet`,
`gofmt`, and cross-compilation for the six 64-bit architectures plus
`js/wasm`, `darwin/arm64` and `windows/amd64`.

## Regenerating the tables

```sh
go run ./cmd/genbidi .
```

This fetches the latest `DerivedBidiClass.txt`, `BidiBrackets.txt`,
`BidiMirroring.txt` and `ArabicShaping.txt` from the Unicode Character Database
and rewrites `bidiclass_table.go`, `bidibrackets_table.go`,
`bidimirror_table.go` and `joining_table.go`.

## Part of the go-opentype pure-Go text stack

`go-opentype/bidi` is the Unicode Bidirectional Algorithm (UBA) layer of a
dependency-free text stack:

- **[opentype](https://github.com/go-opentype/opentype)** — the parsing,
  GSUB/GPOS shaping and rasterising engine.
- **[bidi](https://github.com/go-typeset/bidi)** (this repo) — orders mixed
  left-to-right/right-to-left text into visual order before it is shaped.
- **[shape](https://github.com/go-opentype/shape)** — a HarfBuzz-lite
  complex-script shaper (Arabic, Indic, Hangul, USE, Egyptian
  hieroglyphs, ...) built on `opentype`'s GSUB/GPOS engine; it consumes this
  package's join forms and reordering for right-to-left scripts.
- **[fonts](https://github.com/go-opentype/fonts)** — 46 bundled OFL/BSD
  font families (Latin, non-Latin scripts and CJK), per-family lazily
  `go:embed`-ed, ready to feed to `opentype.Parse`.

## License

BSD-3-Clause. See [LICENSE](LICENSE).
