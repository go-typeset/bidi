// Copyright (c) 2026 the go-opentype/bidi authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package bidi

// Paragraphs implements rule P1: it splits text into paragraphs on characters
// of Bidi_Class B (Paragraph_Separator), returning the paragraphs in order.
//
// Each Paragraph_Separator is kept at the end of the paragraph it terminates,
// and a CR+LF pair (U+000D U+000A) is treated as a single separator. Text after
// the final separator forms a last paragraph; when text ends with a separator
// no trailing empty paragraph is produced. Each returned paragraph can be fed
// independently to [ResolveLevels], [BaseLevel] or [VisualOrder].
func Paragraphs(text []rune) [][]rune {
	var out [][]rune
	start := 0
	for i := 0; i < len(text); i++ {
		if ClassOf(text[i]) != B {
			continue
		}
		end := i + 1
		if text[i] == '\r' && end < len(text) && text[end] == '\n' {
			end++ // CR+LF counts as one separator
		}
		out = append(out, text[start:end])
		start = end
		i = end - 1 // the for-loop's i++ resumes just past the separator
	}
	if start < len(text) {
		out = append(out, text[start:])
	}
	return out
}

// VisualParagraphs runs the full display pipeline on multi-paragraph text: it
// splits text into paragraphs (rule P1), then for each paragraph resolves
// levels, reorders to visual order (L2), keeps combining marks with their base
// (L3) and mirrors right-to-left glyphs (L4). It returns one visual
// (left-to-right) string per paragraph, with characters removed by rule X9
// dropped.
func VisualParagraphs(text string, base Direction) []string {
	paras := Paragraphs([]rune(text))
	out := make([]string, 0, len(paras))
	for _, para := range paras {
		levels, removed, _ := resolve(para, base)
		order := reorderLevels(levels)
		applyMarkReorder(para, levels, order)
		buf := make([]rune, 0, len(para))
		for _, pos := range order {
			if removed[pos] {
				continue
			}
			r := para[pos]
			if levels[pos]%2 == 1 {
				r = Mirror(r)
			}
			buf = append(buf, r)
		}
		out = append(out, string(buf))
	}
	return out
}
