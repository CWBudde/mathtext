package mathtext

import (
	"strconv"
	"strings"
)

func layoutMathTextRun(r Measurer, text string, size float64, fontKey string) mathLayoutBox {
	return layoutMeasuredTextRun(r, text, size, fontKey, true)
}

func layoutDisplayPlainTextRun(r Measurer, text string, size float64, fontKey string) mathLayoutBox {
	return layoutMeasuredTextRun(r, text, size, fontKey, false)
}

func layoutMeasuredTextRun(r Measurer, text string, size float64, fontKey string, applyKerning bool) mathLayoutBox {
	if text == "" {
		return mathLayoutBox{}
	}
	fontKey = mathDisplayOperatorFontKey(text, fontKey)

	// Pixel-exact path: position every glyph individually from matplotlib
	// `_get_info` metrics (mirrors ship() Char/Kern packing). Falls back to the
	// whole-run path when the renderer lacks the FreeType capability (purego).
	if gm, ok := r.(GlyphMeasurer); ok {
		if infos, ok := gm.GlyphRun(text, size, fontKey); ok {
			if box, ok := layoutMathGlyphRun(text, infos, size, fontKey, applyKerning); ok {
				return box
			}
		}
	}

	metrics := r.MeasureText(text, size, fontKey)
	if metrics.W <= 0 {
		metrics.W = float64(len([]rune(text))) * size * 0.5
	}
	if metrics.Ascent <= 0 && metrics.Descent <= 0 {
		metrics.Ascent = size * 0.8
		metrics.Descent = size * 0.2
	}
	return mathLayoutBox{
		runs:    []MathTextLayoutRun{{Text: text, FontSize: size, FontKey: fontKey}},
		Width:   metrics.W,
		Ascent:  metrics.Ascent,
		Descent: metrics.Descent,
	}
}

// layoutMathGlyphRun packs each glyph of text at its exact matplotlib position:
// glyph i renders at cumulative Σ(advance)+Σ(kern), box width = total advance +
// kerns, ascent = max(iceberg), depth = max(height - iceberg) (Char.depth). One
// MathTextLayoutRun is emitted per glyph (baseline-aligned, layout y-down).
func layoutMathGlyphRun(text string, infos []GlyphInfo, size float64, fontKey string, applyKerning bool) (mathLayoutBox, bool) {
	runes := []rune(text)
	if len(infos) != len(runes) || len(runes) == 0 {
		return mathLayoutBox{}, false
	}
	var box mathLayoutBox
	box.runs = make([]MathTextLayoutRun, 0, len(runes))
	x := 0.0
	for i, info := range infos {
		if applyKerning {
			x += info.KernToPrev
		}
		box.runs = append(box.runs, MathTextLayoutRun{
			Text:     string(runes[i]),
			Offset:   Pt{X: x, Y: 0},
			FontSize: size,
			FontKey:  fontKey,
		})
		if info.Iceberg > box.Ascent {
			box.Ascent = info.Iceberg
		}
		if depth := info.Height - info.Iceberg; depth > box.Descent {
			box.Descent = depth
		}
		x += info.Advance
	}
	box.Width = x
	return box, true
}

func mathDisplayOperatorFontKey(text, fontKey string) string {
	if !isMathDisplayOperatorGlyph(text) {
		return fontKey
	}
	switch normalizeMathFontKey(fontKey) {
	case "dejavuserif", "serif":
		return "DejaVu Serif Display"
	default:
		return "DejaVu Sans Display"
	}
}

func normalizeMathFontKey(fontKey string) string {
	fontKey = strings.ToLower(strings.TrimSpace(strings.Trim(fontKey, `"'`)))
	fontKey = strings.ReplaceAll(fontKey, " ", "")
	fontKey = strings.ReplaceAll(fontKey, "_", "")
	return strings.ReplaceAll(fontKey, "-", "")
}

func isMathDisplayOperatorGlyph(text string) bool {
	switch text {
	case "∑", "∏", "∫", "∮":
		return true
	default:
		return false
	}
}

func layoutMathSpace(r Measurer, n mathLayoutNode, size float64, fontKey string) mathLayoutBox {
	return mathLayoutBox{Width: mathQuadWidth(r, size, fontKey) * n.widthEm}
}

func mathQuadWidth(r Measurer, size float64, fontKey string) float64 {
	quad := 0.0
	if r != nil {
		quad = r.MeasureText("m", size, fontKey).W
	}
	if quad <= 0 {
		quad = size * mathSpaceScale
	}
	return quad
}

func layoutMathStyled(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	childFontKey := resolveMathFontKey(fontKey, n, opts)
	return layoutMathNode(r, pointerNode(n.child), size, childFontKey, opts)
}

var mathTextSpacingCommandWidths = map[string]float64{
	",":             0.166,
	":":             0.222,
	";":             0.278,
	"thinspace":     0.166,
	"medspace":      0.222,
	"thickspace":    0.278,
	"negthinspace":  -0.166,
	"negmedspace":   -0.222,
	"negthickspace": -0.278,
	"enspace":       0.5,
	"enskip":        0.5,
	"quad":          1.0,
	"qquad":         2.0,
}

func parseMathSpaceDimension(text string) float64 {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	i := 0
	for i < len(text) {
		c := text[i]
		if c != '+' && c != '-' && c != '.' && (c < '0' || c > '9') {
			break
		}
		i++
	}
	if i == 0 || text[:i] == "+" || text[:i] == "-" || text[:i] == "." {
		return 0
	}
	value, err := strconv.ParseFloat(text[:i], 64)
	if err != nil {
		return 0
	}
	unit := strings.TrimSpace(text[i:])
	switch unit {
	case "", "em":
		return value
	case "ex":
		return value * 0.5
	case "mu":
		return value / 18
	case "pt":
		return value / 10
	default:
		return value
	}
}

func resolveMathFontKey(base string, n mathLayoutNode, opts Options) string {
	request := FontRequest{
		Families: append([]string(nil), n.families...),
		Style:    n.style,
		Weight:   n.weight,
	}
	if opts.FontResolver != nil {
		if resolved := strings.TrimSpace(opts.FontResolver.ResolveMathFontKey(base, request)); resolved != "" {
			return resolved
		}
	}
	if len(request.Families) > 0 {
		return strings.Join(request.Families, ", ")
	}
	return base
}
