package mathtext

import "math"

func layoutMathFrac(r Measurer, num, den mathLayoutNode, size float64, fontKey string, opts Options, rule, display bool, leftDelim, rightDelim string) mathLayoutBox {
	// matplotlib shrinks the numerator/denominator with Node.shrink(), which
	// LINEARLY scales the base-size box metrics by SHRINK_FACTOR (it does NOT
	// re-measure at the smaller size — the hinted iceberg at the shrunk size
	// differs, e.g. '1' iceberg is 10.0 at fs10 → 7.0 scaled, but 8.0 if
	// re-measured at fs7). The glyph bitmap still renders at the shrunk size.
	// So measure at the base size and shrink the box (scaling render FontSize).
	numBox := layoutMathNode(r, num, size, fontKey, opts)
	denBox := layoutMathNode(r, den, size, fontKey, opts)
	if !display { // TEXTSTYLE shrinks once; DISPLAYSTYLE not at all
		numBox = shrinkMathBox(numBox, mathFracShrink)
		denBox = shrinkMathBox(denBox, mathFracShrink)
	}
	thickness := mathUnderlineThickness(r, size, fontKey)
	ruleThickness := thickness
	if !rule {
		ruleThickness = 0
	}
	contentWidth := maxFloat64(numBox.Width, denBox.Width)
	width := contentWidth + 2*thickness // matplotlib trailing Hbox(thickness*2)
	ruleWidth := contentWidth
	// matplotlib centres num/den via HCentered = Hlist([Glue('ss'), x, Glue('ss')]).
	// hlist_out ROUNDS the stretched glue (round(glue_set*cur_glue)), so the left
	// pad is round((contentWidth-boxWidth)/2), not the exact half — a 0.29px
	// centering offset rounds to 0 (e.g. u and v in a 2-row matrix share an ox).
	numX := math.RoundToEven((contentWidth - numBox.Width) / 2)
	denX := math.RoundToEven((contentWidth - denBox.Width) / 2)

	// Faithful port of matplotlib _mathtext.Parser._genfrac. This algorithm is
	// unchanged between 3.8.4 and 3.10.9 (the vendored 3.10.9 source has no
	// axis_height-based variant — _genfrac is still "="-centered). The
	// numerator/rule/denominator stack is Vlist[cnum, Vbox(0,2t), Hrule,
	// Vbox(0,2t), cden] (Hrule height=depth=t/2, so a rule-less fraction keeps a
	// 4t gap), shifted so the rule sits in the middle of "=": shift =
	// cden.height - ("=" center - 3t). Layout space is y-down (negative Y above
	// baseline); matplotlib height=ascent, depth=descent.
	space := thickness * 2.0
	// matplotlib centres the fraction line on "=": shift uses (ymax+ymin)/2 of the
	// "=" glyph. Use exact _get_info bbox when available (GlyphRun), else hinted.
	eqCenter := 0.0
	if gm, ok := r.(GlyphMeasurer); ok {
		if infos, ok := gm.GlyphRun("=", size, fontKey); ok && len(infos) > 0 {
			eqCenter = (infos[0].Ymax + infos[0].Ymin) / 2
		}
	}
	if eqCenter == 0 {
		eq := r.MeasureText("=", size, fontKey)
		eqCenter = (eq.Ascent + eq.Descent) / 2
		if eq.BoundsH > 0 {
			eqCenter = -(eq.BoundsY + eq.BoundsH/2)
		}
	}
	shift := denBox.Ascent - (eqCenter - thickness*3.0)
	denY := shift
	// num→den gap = num.depth + 2*space + ruleAdvance + den.height, where the
	// Hrule advances cur_v by its full height (ruleThickness) BEFORE drawing in
	// vlist_out (a rule-less binom has ruleThickness 0 → 4t gap, a \frac → 5t).
	numY := shift - denBox.Ascent - 2*space - ruleThickness - numBox.Descent
	// The Hrule's pre-advance places its rect top one space below the den top:
	// ruleTop = (denY - den.height) - space (NOT -3t — the earlier −3t put the
	// bar one thickness too high; see vlist_out Box branch).
	ruleTop := shift - denBox.Ascent - space
	vlistHeight := numBox.Ascent + numBox.Descent + 2*space + ruleThickness + denBox.Ascent
	ascent := vlistHeight - shift
	descent := denBox.Descent + shift

	out := mathLayoutBox{
		Width:   width,
		Ascent:  ascent,
		Descent: descent,
	}
	if rule {
		out.rules = append(out.rules, MathTextLayoutRule{
			Rect: Rect{
				Min: Pt{X: 0, Y: ruleTop},
				Max: Pt{X: ruleWidth, Y: ruleTop + ruleThickness},
			},
		})
	}
	out.appendTranslated(numBox, numX, numY)
	out.appendTranslated(denBox, denX, denY)
	if leftDelim == "" && rightDelim == "" {
		return out
	}

	delimAscent := out.Ascent
	delimDescent := out.Descent
	left := layoutMathDelimiter(r, leftDelim, delimAscent, delimDescent, size, fontKey)
	right := layoutMathDelimiter(r, rightDelim, delimAscent, delimDescent, size, fontKey)
	var delimited mathLayoutBox
	x := 0.0
	delimited.appendTranslated(left, x, 0)
	x += left.Width
	delimited.appendTranslated(out, x, 0)
	x += out.Width
	delimited.appendTranslated(right, x, 0)
	x += right.Width
	delimited.Width = x
	delimited.Ascent = maxFloat64(maxFloat64(left.Ascent, out.Ascent), right.Ascent)
	delimited.Descent = maxFloat64(maxFloat64(left.Descent, out.Descent), right.Descent)
	return delimited
}
