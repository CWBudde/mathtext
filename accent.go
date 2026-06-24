package mathtext

import "math"

// mathAccentGlyphRunes maps the letter-named accent commands to the glyph that
// is centred over the nucleus. The runes mirror matplotlib's `_accent_map`
// (_mathtext.py): the spacing/combining accent glyph is rendered as a SEPARATE
// glyph with Accent metrics, not appended to the nucleus as a combining mark.
var mathAccentGlyphRunes = map[string]rune{
	"hat":            '̂', // \circumflexaccent
	"breve":          '̆', // \combiningbreve
	"bar":            '̄', // \combiningoverline (macron)
	"grave":          '̀', // \combininggraveaccent
	"acute":          '́', // \combiningacuteaccent
	"tilde":          '̃', // \combiningtilde
	"dot":            '̇', // \combiningdotabove
	"ddot":           '̈', // \combiningdiaeresis
	"dddot":          '⃛', // \combiningthreedotsabove
	"ddddot":         '⃜', // \combiningfourdotsabove
	"vec":            '⃗', // \combiningrightarrowabove
	"mathring":       '∘', // \circ
	"overrightarrow": '→', // \rightarrow
	"overleftarrow":  '←', // \leftarrow
}

// mathAccentCharRunes maps the single-character accent commands (\^ \~ \' \. \"
// \`) to the same glyph table, matching matplotlib's `_accent_map`.
var mathAccentCharRunes = map[rune]rune{
	'^':  '̂',
	'~':  '̃',
	'\'': '́',
	'.':  '̇',
	'"':  '̈',
	'`':  '̀',
}

// mathWideAccentGlyphRunes maps matplotlib's `_wide_accents` to the glyph that
// is scaled to the nucleus width (matplotlib's AutoWidthChar). DejaVu has no
// sized alternatives, so matplotlib resolves the symbol to its combining mark
// (get_unicode_index) and scales it uniformly — widehat/widetilde/widebar use
// the same combining marks as \hat/\tilde/\bar (U+0302/U+0303/U+0305).
var mathWideAccentGlyphRunes = map[string]rune{
	"widehat":   '̂', // U+0302 COMBINING CIRCUMFLEX ACCENT
	"widetilde": '̃', // U+0303 COMBINING TILDE
	"widebar":   '̅', // U+0305 COMBINING OVERLINE
}

func mathAccentGlyphRune(name string) (rune, bool) {
	r, ok := mathAccentGlyphRunes[name]
	return r, ok
}

func mathWideAccentGlyphRune(name string) (rune, bool) {
	r, ok := mathWideAccentGlyphRunes[name]
	return r, ok
}

// layoutAccentGlyphBox builds a box for one accent glyph with matplotlib's
// `Accent` metrics (_mathtext.py): width = ink width (Xmax-Xmin), height =
// Ymax-Ymin, depth = 0, with the glyph shifted by (-Xmin, +Ymin) so its ink
// rectangle sits at the box origin with its bottom on the baseline.
func layoutAccentGlyphBox(r Measurer, glyph string, size float64, fontKey string) mathLayoutBox {
	if gm, ok := r.(GlyphMeasurer); ok {
		if infos, ok := gm.GlyphRun(glyph, size, fontKey); ok && len(infos) == 1 {
			info := infos[0]
			width := info.Xmax - info.Xmin
			if width <= 0 {
				width = info.Advance
			}
			height := info.Ymax - info.Ymin
			if height <= 0 {
				height = info.Height
			}
			return mathLayoutBox{
				runs: []MathTextLayoutRun{{
					Text:     glyph,
					Offset:   Pt{X: -info.Xmin, Y: info.Ymin},
					FontSize: size,
					FontKey:  fontKey,
				}},
				Width:   width,
				Ascent:  height,
				Descent: 0,
			}
		}
	}
	// Fallback for renderers without per-glyph metrics (purego/WASM): approximate
	// the ink box from whole-run measurement; the accent sits on the baseline.
	m := r.MeasureText(glyph, size, fontKey)
	width := m.W
	if width <= 0 {
		width = size * 0.5
	}
	height := m.Ascent
	if m.BoundsH > 0 {
		height = m.BoundsH
	}
	if height <= 0 {
		height = size * 0.5
	}
	return mathLayoutBox{
		runs:    []MathTextLayoutRun{{Text: glyph, FontSize: size, FontKey: fontKey}},
		Width:   width,
		Ascent:  height,
		Descent: 0,
	}
}

// autoWidthAccentBox ports matplotlib's AutoWidthChar with char_class=Accent for
// the DejaVu Sans fontset. get_sized_alternatives_for_symbol returns the size-0
// DejaVu Sans combining mark followed by the STIX sized-symbol variants; the
// first whose ink width reaches the target nucleus width is selected, then the
// glyph is scaled so its ink width matches the target exactly. This yields a
// wide-but-thin accent (vs. uniformly scaling the small base glyph, which would
// be far too bold).
func autoWidthAccentBox(r Measurer, glyph string, target, size float64) mathLayoutBox {
	candidates := []mathDelimiterGlyph{
		{text: glyph, fontKey: mathAutoHeightBaseFontKey},
		{text: glyph, fontKey: "STIXSizeOneSym"},
		{text: glyph, fontKey: "STIXSizeTwoSym"},
		{text: glyph, fontKey: "STIXSizeThreeSym"},
		{text: glyph, fontKey: "STIXSizeFourSym"},
		{text: glyph, fontKey: "STIXSizeFiveSym"},
	}
	selected := candidates[len(candidates)-1]
	selectedWidth := 0.0
	for _, c := range candidates {
		w := layoutAccentGlyphBox(r, c.text, size, c.fontKey).Width
		if w <= 0 {
			continue
		}
		selected = c
		selectedWidth = w
		if w >= target {
			break
		}
	}
	if selectedWidth <= 0 {
		return layoutAccentGlyphBox(r, glyph, size, mathAutoHeightBaseFontKey)
	}
	accentSize := size * target / selectedWidth
	if accentSize <= 0 {
		accentSize = size
	}
	return layoutAccentGlyphBox(r, selected.text, accentSize, selected.fontKey)
}

// layoutMathAccent is a faithful port of matplotlib's `Parser.accent`
// (_mathtext.py): the accent glyph is centred over the nucleus via
// HCentered([Hbox(W/4), accent]).hpack(W) and stacked above it with a
// 2*thickness gap. Wide accents (\widehat/\widetilde/\widebar) scale the glyph
// uniformly to the nucleus width (matplotlib's AutoWidthChar).
func layoutMathAccent(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	nucleus := layoutMathNode(r, pointerNode(n.child), size, fontKey, opts)
	width := nucleus.Width
	thickness := mathUnderlineThickness(r, size, fontKey)

	var accentBox mathLayoutBox
	if n.accentWide {
		accentBox = autoWidthAccentBox(r, n.accent, width, size)
	} else {
		accentBox = layoutAccentGlyphBox(r, n.accent, size, fontKey)
		for i := 0; i < n.accentRings; i++ {
			accentBox = shrinkMathBox(accentBox, mathFracShrink)
		}
	}

	// matplotlib HCentered([Hbox(width/4), accent]).hpack(width, 'exactly'):
	// centre the [pad, accent] block within the nucleus width. hlist_out rounds
	// the stretched centring glue, so round the left pad (see layoutMathFrac).
	blockWidth := width/4 + accentBox.Width
	leftPad := math.RoundToEven((width - blockWidth) / 2)
	accentX := leftPad + width/4

	gap := thickness * 2.0

	var out mathLayoutBox
	out.appendTranslated(nucleus, 0, 0)
	out.Width = width
	out.Ascent = nucleus.Ascent
	out.Descent = nucleus.Descent
	if n.accentUnder {
		// Best-effort \underbrace: place the (wide) accent below the nucleus.
		y := nucleus.Descent + gap + accentBox.Ascent
		out.appendTranslated(accentBox, accentX, y)
		out.Descent = maxFloat64(out.Descent, y+accentBox.Descent)
		return out
	}
	y := -(nucleus.Ascent + gap + accentBox.Descent)
	out.appendTranslated(accentBox, accentX, y)
	out.Ascent = maxFloat64(out.Ascent, -y+accentBox.Ascent)
	return out
}

// layoutMathOverline ports matplotlib's `Parser.overline` (_mathtext.py): a
// full-width horizontal rule of one underline thickness is placed above the body
// with a 2*thickness + clearance gap, where clearance = fontsize*dpi/(100*12).
func layoutMathOverline(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	body := layoutMathNode(r, pointerNode(n.child), size, fontKey, opts)
	thickness := mathUnderlineThickness(r, size, fontKey)
	clearance := mathFontSizePixels(r, size, fontKey) * 72.0 / 1200.0

	var out mathLayoutBox
	out.appendTranslated(body, 0, 0)
	out.Width = body.Width
	out.Descent = body.Descent
	// The bar sits a 2*thickness gap above the body (rule top at body.Ascent+3t);
	// the clearance is padding ABOVE the bar, extending the box ascent so the
	// va-centered placement matches matplotlib without moving the bar itself.
	out.Ascent = body.Ascent + thickness*3.0 + clearance

	ruleTop := -(body.Ascent + thickness*3.0)
	out.rules = append(out.rules, MathTextLayoutRule{
		Rect: Rect{
			Min: Pt{X: 0, Y: ruleTop},
			Max: Pt{X: body.Width, Y: ruleTop + thickness},
		},
	})
	return out
}

// layoutMathStack ports matplotlib's `Parser._genset` (\overset/\underset, plus
// the \stackrel extension): the (shrunk) annotation is centred over or under the
// body with a 3*thickness gap; both are horizontally centred to the max width.
func layoutMathStack(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	body := layoutMathNode(r, pointerNode(n.base), size, fontKey, opts)
	thickness := mathUnderlineThickness(r, size, fontKey)
	gap := thickness * 3.0

	width := body.Width
	var top, bottom mathLayoutBox
	if n.stackTop != nil {
		top = shrinkMathBox(layoutMathNode(r, *n.stackTop, size, fontKey, opts), mathFracShrink)
		width = maxFloat64(width, top.Width)
	}
	if n.stackBottom != nil {
		bottom = shrinkMathBox(layoutMathNode(r, *n.stackBottom, size, fontKey, opts), mathFracShrink)
		width = maxFloat64(width, bottom.Width)
	}

	var out mathLayoutBox
	out.Width = width
	bodyX := math.RoundToEven((width - body.Width) / 2)
	out.appendTranslated(body, bodyX, 0)
	out.Ascent = body.Ascent
	out.Descent = body.Descent

	if n.stackTop != nil {
		x := math.RoundToEven((width - top.Width) / 2)
		y := -(body.Ascent + gap + top.Descent)
		out.appendTranslated(top, x, y)
		out.Ascent = maxFloat64(out.Ascent, -y+top.Ascent)
	}
	if n.stackBottom != nil {
		x := math.RoundToEven((width - bottom.Width) / 2)
		y := body.Descent + gap + bottom.Ascent
		out.appendTranslated(bottom, x, y)
		out.Descent = maxFloat64(out.Descent, y+bottom.Descent)
	}
	return out
}

// layoutMathSubstack ports matplotlib's `Parser.substack` (_mathtext.py): each
// \\-separated line is centred to the max width and stacked with a 2*thickness
// gap. The Vlist baseline is the bottom line (matplotlib's default vpack).
func layoutMathSubstack(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	if len(n.children) == 0 {
		return mathLayoutBox{}
	}
	thickness := mathUnderlineThickness(r, size, fontKey)
	gap := thickness * 2.0

	lines := make([]mathLayoutBox, len(n.children))
	width := 0.0
	for i, line := range n.children {
		lines[i] = layoutMathNode(r, line, size, fontKey, opts)
		width = maxFloat64(width, lines[i].Width)
	}

	var out mathLayoutBox
	out.Width = width
	last := len(lines) - 1
	bottomX := math.RoundToEven((width - lines[last].Width) / 2)
	out.appendTranslated(lines[last], bottomX, 0)
	out.Ascent = lines[last].Ascent
	out.Descent = lines[last].Descent

	top := -lines[last].Ascent
	for i := last - 1; i >= 0; i-- {
		line := lines[i]
		yb := top - gap - line.Descent
		x := math.RoundToEven((width - line.Width) / 2)
		out.appendTranslated(line, x, yb)
		out.Ascent = maxFloat64(out.Ascent, -yb+line.Ascent)
		top = yb - line.Ascent
	}
	return out
}
