package mathtext

import "strings"

// LayoutMathText parses and lays out one MathText expression without requiring
// dollar delimiters. It supports the same fallback command set as displayed
// text normalization plus baseline-shifted scripts, stacked fractions, and
// square-root vincula.
func LayoutMathText(m Measurer, expr string, size float64, fontKey string, opts Options) (MathTextLayout, bool) {
	if m == nil || strings.TrimSpace(expr) == "" || size <= 0 {
		return MathTextLayout{}, false
	}
	expr = strings.ReplaceAll(expr, `\\`, `\`)
	cacheKey, useLayoutCache := opts.layoutCacheKey("math", expr, size, fontKey)
	if useLayoutCache {
		if layout, ok := opts.Cache.layout(cacheKey); ok {
			return layout, true
		}
	}
	node := parseMathLayoutNode(expr, opts.Cache)
	box := layoutMathNode(m, node, size, fontKey, opts)
	if box.Width <= 0 && len(box.runs) == 0 && len(box.rules) == 0 {
		return MathTextLayout{}, false
	}
	layout := MathTextLayout{
		Runs:    box.runs,
		Rules:   box.rules,
		Width:   box.Width,
		Ascent:  box.Ascent,
		Descent: box.Descent,
		Height:  box.Ascent + box.Descent,
	}
	if useLayoutCache {
		opts.Cache.storeLayout(cacheKey, layout)
	}
	return cloneLayout(layout), true
}

// LayoutDisplay lays out display text with either a single full $...$
// expression or mixed plain text and inline $...$ MathText segments.
func LayoutDisplay(m Measurer, text string, size float64, fontKey string, opts Options) (MathTextLayout, bool) {
	if expr, ok := FullExpression(text); ok {
		return LayoutMathText(m, expr, size, fontKey, opts)
	}

	cacheKey, useLayoutCache := opts.layoutCacheKey("display", text, size, fontKey)
	if useLayoutCache {
		if layout, ok := opts.Cache.layout(cacheKey); ok {
			return layout, true
		}
	}

	segments, hasMath, ok := SplitDisplaySegments(text)
	if !ok || !hasMath {
		return MathTextLayout{}, false
	}

	var out mathLayoutBox
	x := 0.0
	for _, segment := range segments {
		var child mathLayoutBox
		if segment.IsMath {
			layout, ok := LayoutMathText(m, segment.Text, size, fontKey, opts)
			if !ok {
				return MathTextLayout{}, false
			}
			child = mathLayoutBox{
				runs:    append([]MathTextLayoutRun(nil), layout.Runs...),
				rules:   append([]MathTextLayoutRule(nil), layout.Rules...),
				Width:   layout.Width,
				Ascent:  layout.Ascent,
				Descent: layout.Descent,
			}
		} else {
			child = layoutDisplayPlainTextRun(m, displayTextCommandReplacer.Replace(segment.Text), size, fontKey)
		}
		if child.Width <= 0 && len(child.runs) == 0 && len(child.rules) == 0 {
			continue
		}
		out.appendTranslated(child, x, 0)
		x += child.Width
		out.Ascent = maxFloat64(out.Ascent, child.Ascent)
		out.Descent = maxFloat64(out.Descent, child.Descent)
	}

	if x <= 0 && len(out.runs) == 0 && len(out.rules) == 0 {
		return MathTextLayout{}, false
	}

	layout := MathTextLayout{
		Runs:    out.runs,
		Rules:   out.rules,
		Width:   x,
		Ascent:  out.Ascent,
		Descent: out.Descent,
		Height:  out.Ascent + out.Descent,
	}
	if useLayoutCache {
		opts.Cache.storeLayout(cacheKey, layout)
	}
	return cloneLayout(layout), true
}

func (o Options) layoutCacheKey(kind, text string, size float64, fontKey string) (layoutCacheKey, bool) {
	if o.Cache == nil || strings.TrimSpace(o.MeasurementKey) == "" {
		return layoutCacheKey{}, false
	}
	return layoutCacheKey{
		kind:           kind,
		text:           text,
		size:           size,
		fontKey:        fontKey,
		measurementKey: o.MeasurementKey,
	}, true
}

func layoutMathNode(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	switch n.kind {
	case mathLayoutText:
		return layoutMathTextRun(r, n.text, size, fontKey)
	case mathLayoutList:
		return layoutMathList(r, n.children, size, fontKey, opts)
	case mathLayoutScript:
		return layoutMathScript(r, n, size, fontKey, opts)
	case mathLayoutFrac:
		return layoutMathFrac(r, pointerNode(n.num), pointerNode(n.den), size, fontKey, opts, n.fracRule, n.fracDisp, n.left, n.right)
	case mathLayoutSqrt:
		return layoutMathSqrt(r, pointerNode(n.radicand), n.index, size, fontKey, opts)
	case mathLayoutSpace:
		return layoutMathSpace(r, n, size, fontKey)
	case mathLayoutStyled:
		return layoutMathStyled(r, n, size, fontKey, opts)
	case mathLayoutFence:
		return layoutMathFence(r, n, size, fontKey, opts)
	case mathLayoutMatrix:
		return layoutMathMatrix(r, n, size, fontKey, opts)
	default:
		return mathLayoutBox{}
	}
}

func layoutMathList(r Measurer, children []mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	var out mathLayoutBox
	x := 0.0
	for _, child := range children {
		box := layoutMathNode(r, child, size, fontKey, opts)
		out.appendTranslated(box, x, 0)
		x += box.Width
		if box.Ascent > out.Ascent {
			out.Ascent = box.Ascent
		}
		if box.Descent > out.Descent {
			out.Descent = box.Descent
		}
	}
	out.Width = x
	return out
}

// FontConstantsBase defaults from matplotlib's lib/matplotlib/_mathtext.py.
// These sub/superscript constants are unchanged between matplotlib 3.8.4 and
// 3.10.9 (verified against the vendored 3.10.9 source), and DejaVuSansFontConstants
// is literally `pass` in both versions, so DejaVu Sans uses these base values.
// They are multiples of the scaled x-height (sup/sub shifts) or the pixel font
// size (underline). The reference images (now generated under 3.10.9) match.
const (
	// matplotlib FontConstantsBase sub/superscript constants (DejaVu Sans = base).
	mathScriptDelta         = 0.025 // delta
	mathScriptDeltaSlanted  = 0.2   // delta_slanted
	mathScriptDeltaIntegral = 0.1   // delta_integral
	mathScriptSpace         = 0.05  // script_space
	mathScriptSubdrop       = 0.4   // subdrop
	mathScriptSup1          = 0.7   // sup1
	mathScriptSub1          = 0.3   // sub1
	mathScriptSub2          = 0.5   // sub2

	// Design-em ratio for DejaVu Sans' x-height (xHeight 1120 / unitsPerEm 2048),
	// used only to recover the pixel font size for underline thickness. NOTE:
	// matplotlib's get_xheight does NOT use this ratio for DejaVu (which has no
	// PCLT table) — it returns the iceberg (top ink extent) of glyph 'x'. See
	// mathXHeight/mathFontSizePixels for how this cancels to the iceberg value.
	mathDejaVuSansXHeight = 1120.0 / 2048.0
	mathUnderlineRatio    = 0.75 / 12.0 // get_underline_thickness (hardcoded)

	// SHRINK_FACTOR: TeX style step shrinking numerator/denominator and scripts.
	mathFracShrink = 0.70
)

// mathIceberg returns matplotlib's "iceberg" for a glyph: the top ink extent
// above the baseline (FreeType horiBearingY), in device pixels. Layout bounds
// are y-down with negative Y above the baseline, so the iceberg is -BoundsY.
// Returns 0 when the renderer cannot supply ink bounds.
func mathIceberg(r Measurer, text string, size float64, fontKey string) float64 {
	if r == nil {
		return 0
	}
	// Pixel-exact: iceberg = max(horiBearingY/64) over the glyphs (matplotlib
	// `_get_info` iceberg / get_xheight). Falls back to hinted ink bounds.
	if gm, ok := r.(GlyphMeasurer); ok {
		if infos, ok := gm.GlyphRun(text, size, fontKey); ok && len(infos) > 0 {
			ice := 0.0
			for _, info := range infos {
				if info.Iceberg > ice {
					ice = info.Iceberg
				}
			}
			if ice > 0 {
				return ice
			}
		}
	}
	m := r.MeasureText(text, size, fontKey)
	if m.BoundsH <= 0 {
		return 0
	}
	if ice := -m.BoundsY; ice > 0 {
		return ice
	}
	return 0
}

// mathXHeight is matplotlib TruetypeFonts.get_xheight for DejaVu Sans. DejaVu
// has no PCLT table, so matplotlib uses the "poor man's xHeight" = the iceberg
// (top ink extent above the baseline) of glyph 'x'. We return that directly.
func mathXHeight(r Measurer, size float64, fontKey string) float64 {
	if xh := mathIceberg(r, "x", size, fontKey); xh > 0 {
		return xh
	}
	return mathFontSizePixels(r, size, fontKey) * mathDejaVuSansXHeight
}

// mathFontSizePixels recovers matplotlib's box-model unit — the font size in
// device pixels (fontsize*dpi/72) — which is needed for the (font-independent)
// underline thickness. The Measurer interface exposes only the point size, so
// we recover the pixel size from the measured x-height of 'x' (its iceberg)
// divided by DejaVu Sans' design-em x-height ratio (1120/2048). For DejaVu 'x'
// (which sits on the baseline) this iceberg equals the full ink height.
func mathFontSizePixels(r Measurer, size float64, fontKey string) float64 {
	// Exact when the renderer exposes DPI (matplotlib fontsize*dpi/72). The
	// x-height reconstruction below is an approximation used only as a fallback
	// (purego/mock renderers) — its ratio drifts with the autohinter per size,
	// which is why fraction-bar/thickness positions were off without exact DPI.
	if dp, ok := r.(DPIMeasurer); ok {
		if dpi := dp.DPI(); dpi > 0 {
			return size * dpi / 72.0
		}
	}
	if xh := mathIceberg(r, "x", size, fontKey); xh > 0 {
		return xh / mathDejaVuSansXHeight
	}
	return mathQuadWidth(r, size, fontKey)
}

// mathUnderlineThickness is matplotlib get_underline_thickness: hardcoded to
// (0.75/12)*fontsize*dpi/72 because upstream found font underline metrics too
// unreliable. The base unit for fraction bars, radical rules, and script gaps.
func mathUnderlineThickness(r Measurer, size float64, fontKey string) float64 {
	return mathFontSizePixels(r, size, fontKey) * mathUnderlineRatio
}
