# bidi

[![CI](https://github.com/go-opentype/bidi/actions/workflows/ci.yml/badge.svg)](https://github.com/go-opentype/bidi/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-opentype/bidi.svg)](https://pkg.go.dev/github.com/go-opentype/bidi)
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
go get github.com/go-opentype/bidi
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-opentype/bidi"
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

## API

| Symbol | Purpose |
| --- | --- |
| `ClassOf(r rune) Class` | Bidi_Class of a rune |
| `Class` enum | `L R AL EN ES ET AN CS NSM BN B S WS ON LRE RLE LRO RLO PDF LRI RLI FSI PDI` |
| `ResolveLevels(text []rune, base Direction) []Level` | resolved embedding level per rune |
| `BaseLevel(text []rune, base Direction) Level` | paragraph level (rules P2/P3) |
| `Reorder(text []rune, levels []Level) []int` | rule L2 visual-order permutation |
| `VisualOrder(text string, base Direction) string` | resolve + reorder convenience |
| `Direction` | `LeftToRight`, `RightToLeft`, `Auto` |
| `Level` | embedding level (even = LTR, odd = RTL) |

## Implemented vs deferred

**Implemented** — the algorithm runs through rule **L2**, the full extent
covered by the Unicode conformance file `BidiCharacterTest.txt`:

- **P2, P3** base paragraph level from the first strong character.
- **X1–X8** explicit embeddings *and* isolates (with overflow handling and the
  directional status stack); **X9** removal of the deprecated formatting
  characters and `BN`; **X10** isolating run sequences with `sos`/`eos`.
- **W1–W7** weak types.
- **N0** paired-bracket resolution (BD16, incl. the U+2329/U+232A canonical
  equivalence), **N1–N2** neutral types.
- **I1, I2** implicit levels.
- **L1** separator / trailing-whitespace reset, **L2** reordering.

**Deferred** (out of scope for a bidi engine):

- **L3** (combining marks) and **L4** (glyph mirroring of paired brackets and
  other mirrored characters) — these belong to the rendering / shaping stage, so
  `VisualOrder` does not substitute mirrored glyphs.
- Arabic cursive **shaping / joining** — the job of a shaper, not bidi.
- **P1** paragraph splitting — the caller drives it; the API operates per
  paragraph (an inline `Paragraph_Separator` is still handled by X8/L1).

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

This fetches the latest `DerivedBidiClass.txt` and `BidiBrackets.txt` from the
Unicode Character Database and rewrites `bidiclass_table.go` and
`bidibrackets_table.go`.

## License

BSD-3-Clause. See [LICENSE](LICENSE).
