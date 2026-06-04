package mathtext

func layoutMathFence(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	segments := fenceSegments(n)
	if len(segments) == 0 {
		return mathLayoutBox{}
	}

	bodyBoxes := make([]mathLayoutBox, len(segments))
	bodyAscent := 0.0
	bodyDescent := 0.0
	for i, segment := range segments {
		bodyBoxes[i] = layoutMathNode(r, segment, size, fontKey, opts)
		if bodyBoxes[i].Ascent > bodyAscent {
			bodyAscent = bodyBoxes[i].Ascent
		}
		if bodyBoxes[i].Descent > bodyDescent {
			bodyDescent = bodyBoxes[i].Descent
		}
	}

	left := layoutMathDelimiter(r, n.left, bodyAscent, bodyDescent, size, fontKey)
	middles := make([]mathLayoutBox, len(n.middles))
	for i, middle := range n.middles {
		middles[i] = layoutMathDelimiter(r, middle, bodyAscent, bodyDescent, size, fontKey)
	}
	right := layoutMathDelimiter(r, n.right, bodyAscent, bodyDescent, size, fontKey)

	var out mathLayoutBox
	x := 0.0
	out.appendTranslated(left, x, 0)
	x += left.Width
	for i, body := range bodyBoxes {
		out.appendTranslated(body, x, 0)
		x += body.Width
		if i < len(middles) {
			out.appendTranslated(middles[i], x, 0)
			x += middles[i].Width
		}
	}
	out.appendTranslated(right, x, 0)
	x += right.Width
	out.Width = x
	out.Ascent = maxFloat64(maxFloat64(left.Ascent, bodyAscent), right.Ascent)
	out.Descent = maxFloat64(maxFloat64(left.Descent, bodyDescent), right.Descent)
	for _, middle := range middles {
		out.Ascent = maxFloat64(out.Ascent, middle.Ascent)
		out.Descent = maxFloat64(out.Descent, middle.Descent)
	}
	return out
}

func layoutMathDelimiter(r Measurer, delim string, targetAscent, targetDescent, size float64, fontKey string) mathLayoutBox {
	if delim == "" {
		return mathLayoutBox{}
	}
	if targetAscent <= 0 && targetDescent <= 0 {
		targetAscent = size * 0.8
		targetDescent = size * 0.2
	}
	switch delim {
	case "|", "‖":
		return layoutMathVerticalRuleDelimiter(delim, targetAscent, targetDescent, size)
	default:
		if candidates := mathSizedDelimiterGlyphs(delim); len(candidates) > 0 {
			return layoutMathSizedDelimiter(r, candidates, targetAscent, targetDescent, size)
		}
		delimiterSize := maxFloat64(size*1.1, (targetAscent+targetDescent)*mathGlyphDelimiterScale)
		return centerMathDelimiterBox(layoutMathTextRun(r, delim, delimiterSize, mathDelimiterFontKey()), targetAscent, targetDescent)
	}
}

func layoutMathSizedDelimiter(r Measurer, candidates []mathDelimiterGlyph, targetAscent, targetDescent, size float64) mathLayoutBox {
	return autoHeightChar(r, candidates, targetAscent, targetDescent, size)
}

// mathAutoHeightBaseFontKey is the size-0 (unscaled) variant font in the DejaVu
// Sans fontset's sized-alternatives list. matplotlib never scales the size-0
// glyph when larger variants exist.
const mathAutoHeightBaseFontKey = "DejaVu Sans"

// autoHeightChar ports matplotlib _mathtext.AutoHeightChar for the DejaVu Sans
// fontset (used for auto-sized delimiters and the sqrt radical). candidates must
// be ordered smallest→largest with candidates[0] the size-0 DejaVu Sans glyph.
// It selects the first variant whose ink height+depth reaches targetTotal minus
// the 0.2*xHeight slack (else the largest), then — unless the size-0 glyph was
// chosen and alternatives exist — scales it by factor = targetTotal/(h+d) and
// shifts it down by shift_amount = targetDescent - char.depth.
func autoHeightChar(r Measurer, candidates []mathDelimiterGlyph, targetAscent, targetDescent, size float64) mathLayoutBox {
	if len(candidates) == 0 {
		return mathLayoutBox{}
	}
	targetTotal := targetAscent + targetDescent
	if targetTotal <= 0 {
		targetTotal = size
	}
	threshold := targetTotal - 0.2*mathXHeight(r, size, mathAutoHeightBaseFontKey)

	selected := candidates[len(candidates)-1]
	selectedTotal := 0.0
	for _, candidate := range candidates {
		box := layoutMathTextRun(r, candidate.text, size, candidate.fontKey)
		total := box.Ascent + box.Descent
		if total <= 0 {
			continue
		}
		selected = candidate
		selectedTotal = total
		if total >= threshold {
			break
		}
	}

	// size-0 (DejaVu Sans) is rendered unscaled, baseline-aligned (shift 0).
	if selected.fontKey == mathAutoHeightBaseFontKey && len(candidates) > 1 {
		return layoutMathTextRun(r, selected.text, size, selected.fontKey)
	}
	if selectedTotal <= 0 {
		selectedTotal = size
	}
	fontsize := size * targetTotal / selectedTotal
	if fontsize <= 0 {
		fontsize = size
	}
	box := layoutMathTextRun(r, selected.text, fontsize, selected.fontKey)
	return shiftMathBoxDown(box, targetDescent-box.Descent)
}

// shrinkMathBox ports matplotlib Node.shrink(): it LINEARLY scales a laid-out
// box's geometry (run offsets, rule rects, width, ascent, depth) by factor and
// multiplies each run's render FontSize by factor. This deliberately differs
// from re-measuring at the smaller size — matplotlib's box model positions
// shrunk content with metrics scaled from the base measurement, while the glyph
// bitmaps still rasterize at the shrunk fontsize.

func mathSizedDelimiterGlyphs(delim string) []mathDelimiterGlyph {
	switch delim {
	case "(", ")", "[", "]", "{", "}", "⌊", "⌋", "⌈", "⌉", "⟨", "⟩":
		return []mathDelimiterGlyph{
			{text: delim, fontKey: mathAutoHeightBaseFontKey},
			stixSizeGlyph(1, delim),
			stixSizeGlyph(2, delim),
			stixSizeGlyph(3, delim),
			stixSizeGlyph(4, delim),
		}
	default:
		return nil
	}
}

func stixSizeGlyph(size int, text string) mathDelimiterGlyph {
	switch size {
	case 1:
		return mathDelimiterGlyph{text: text, fontKey: "STIXSizeOneSym"}
	case 2:
		return mathDelimiterGlyph{text: text, fontKey: "STIXSizeTwoSym"}
	case 3:
		return mathDelimiterGlyph{text: text, fontKey: "STIXSizeThreeSym"}
	case 4:
		return mathDelimiterGlyph{text: text, fontKey: "STIXSizeFourSym"}
	default:
		return mathDelimiterGlyph{text: text, fontKey: "STIXSizeFiveSym"}
	}
}

func layoutMathVerticalRuleDelimiter(delim string, targetAscent, targetDescent, size float64) mathLayoutBox {
	thickness := maxFloat64(size*0.065, 1.0)
	pad := size * 0.04
	targetAscent *= mathRuleDelimiterScale
	targetDescent *= mathRuleDelimiterScale
	top := -targetAscent - pad
	bottom := targetDescent + pad
	width := maxFloat64(size*0.18, thickness*2)
	centers := []float64{width / 2}
	if delim == "‖" {
		width = maxFloat64(size*0.30, thickness*4)
		gap := maxFloat64(thickness*1.4, size*0.06)
		centers = []float64{width/2 - gap/2, width/2 + gap/2}
	}
	rules := make([]MathTextLayoutRule, 0, len(centers))
	for _, center := range centers {
		rules = append(rules, MathTextLayoutRule{
			Rect: Rect{
				Min: Pt{X: center - thickness/2, Y: top},
				Max: Pt{X: center + thickness/2, Y: bottom},
			},
		})
	}
	return mathLayoutBox{
		rules:   rules,
		Width:   width,
		Ascent:  -top,
		Descent: bottom,
	}
}

func layoutMathBracketDelimiter(delim string, targetAscent, targetDescent, size float64) mathLayoutBox {
	thickness := maxFloat64(size*0.065, 1.0)
	pad := size * 0.04
	targetAscent *= mathRuleDelimiterScale
	targetDescent *= mathRuleDelimiterScale
	top := -targetAscent - pad
	bottom := targetDescent + pad
	width := maxFloat64(size*0.28, thickness*4)
	left := delim == "[" || delim == "⌊" || delim == "⌈"
	topCap := delim == "[" || delim == "]" || delim == "⌈" || delim == "⌉"
	bottomCap := delim == "[" || delim == "]" || delim == "⌊" || delim == "⌋"
	x0 := 0.0
	x1 := thickness
	if !left {
		x0 = width - thickness
		x1 = width
	}
	rules := []MathTextLayoutRule{{
		Rect: Rect{
			Min: Pt{X: x0, Y: top},
			Max: Pt{X: x1, Y: bottom},
		},
	}}
	if topCap {
		rules = append(rules, MathTextLayoutRule{
			Rect: Rect{
				Min: Pt{X: 0, Y: top},
				Max: Pt{X: width, Y: top + thickness},
			},
		})
	}
	if bottomCap {
		rules = append(rules, MathTextLayoutRule{
			Rect: Rect{
				Min: Pt{X: 0, Y: bottom - thickness},
				Max: Pt{X: width, Y: bottom},
			},
		})
	}
	return mathLayoutBox{
		rules:   rules,
		Width:   width,
		Ascent:  -top,
		Descent: bottom,
	}
}

// mathSqrtRadicalGlyphs lists the auto-height variants of the radical sign for
// the DejaVu Sans fontset: size-0 DejaVu Sans, then STIX size variants 1-3
// (matplotlib drops the largest STIX radical: alternatives[:-1]).
func mathSqrtRadicalGlyphs() []mathDelimiterGlyph {
	return []mathDelimiterGlyph{
		{text: "√", fontKey: mathAutoHeightBaseFontKey},
		{text: "√", fontKey: "STIXSizeOneSym"},
		{text: "√", fontKey: "STIXSizeTwoSym"},
		{text: "√", fontKey: "STIXSizeThreeSym"},
	}
}
