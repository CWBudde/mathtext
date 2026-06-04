package mathtext

import "math"

func layoutMathMatrix(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	if len(n.rows) == 0 {
		return mathLayoutBox{}
	}

	cellBoxes := make([][]mathLayoutBox, len(n.rows))
	numCols := 0
	for i, row := range n.rows {
		cellBoxes[i] = make([]mathLayoutBox, len(row))
		if len(row) > numCols {
			numCols = len(row)
		}
		for j, cell := range row {
			cellBoxes[i][j] = layoutMathNode(r, cell, size, fontKey, opts)
		}
	}
	if numCols == 0 {
		return mathLayoutBox{}
	}

	colWidths := make([]float64, numCols)
	rowAscents := make([]float64, len(n.rows))
	rowDescents := make([]float64, len(n.rows))
	for i, row := range cellBoxes {
		for j, cell := range row {
			if cell.Width > colWidths[j] {
				colWidths[j] = cell.Width
			}
			if cell.Ascent > rowAscents[i] {
				rowAscents[i] = cell.Ascent
			}
			if cell.Descent > rowDescents[i] {
				rowDescents[i] = cell.Descent
			}
		}
		if rowAscents[i] == 0 && rowDescents[i] == 0 {
			rowAscents[i] = size * 0.5
			rowDescents[i] = size * 0.3
		}
	}

	colGap := size * 0.6
	rowGap := size * 0.4
	bodyWidth := 0.0
	for i, width := range colWidths {
		bodyWidth += width
		if i > 0 {
			bodyWidth += colGap
		}
	}
	bodyHeight := 0.0
	for i := range n.rows {
		bodyHeight += rowAscents[i] + rowDescents[i]
		if i > 0 {
			bodyHeight += rowGap
		}
	}

	left := layoutMathDelimiter(r, n.left, bodyHeight/2, bodyHeight/2, size, fontKey)
	right := layoutMathDelimiter(r, n.right, bodyHeight/2, bodyHeight/2, size, fontKey)
	leftGap := 0.0
	rightGap := 0.0
	if left.Width > 0 {
		leftGap = size * 0.18
	}
	if right.Width > 0 {
		rightGap = size * 0.18
	}

	var out mathLayoutBox
	x := 0.0
	out.appendTranslated(left, x, 0)
	x += left.Width
	if left.Width > 0 {
		x += leftGap
	}

	top := -bodyHeight / 2
	for i, row := range cellBoxes {
		baselineY := top + rowAscents[i]
		cellX := x
		for j := 0; j < numCols; j++ {
			var cell mathLayoutBox
			if j < len(row) {
				cell = row[j]
			}
			cellOffsetX := cellX + (colWidths[j]-cell.Width)/2
			out.appendTranslated(cell, cellOffsetX, baselineY)
			cellX += colWidths[j] + colGap
		}
		top += rowAscents[i] + rowDescents[i] + rowGap
	}
	x += bodyWidth
	if right.Width > 0 {
		x += rightGap
	}
	out.appendTranslated(right, x, 0)
	out.Width = left.Width + leftGap + bodyWidth + rightGap + right.Width
	out.Ascent = bodyHeight / 2
	out.Descent = bodyHeight / 2
	if left.Ascent > out.Ascent {
		out.Ascent = left.Ascent
	}
	if right.Ascent > out.Ascent {
		out.Ascent = right.Ascent
	}
	if left.Descent > out.Descent {
		out.Descent = left.Descent
	}
	if right.Descent > out.Descent {
		out.Descent = right.Descent
	}
	return out
}

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

func layoutMathSqrt(r Measurer, radicand mathLayoutNode, index *mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	thickness := mathUnderlineThickness(r, size, fontKey)
	radicandBox := layoutMathNode(r, radicand, size, fontKey, opts)
	// matplotlib sqrt(): the radical (check) is an AutoHeightChar sized to the
	// body height + 5*thickness (extra so it doesn't look cramped), body depth.
	targetHeight := radicandBox.Ascent + 5*thickness
	targetDepth := radicandBox.Descent
	root := autoHeightChar(r, mathSqrtRadicalGlyphs(), targetHeight, targetDepth, size)

	// matplotlib re-derives height/depth from the (possibly scaled) radical:
	//   height = check.height - check.shift_amount  (= root.Ascent)
	//   depth  = check.depth  + check.shift_amount  (= root.Descent)
	// then builds rightside = Vlist[Hrule, Glue('fill'), padded_body] and
	// vpacks it to EXACTLY height + extra, where
	//   extra = (fontsize*dpi)/(100*12) = mathFontSizePixels * 72/1200.
	bodyHeight := root.Ascent
	extra := mathFontSizePixels(r, size, fontKey) * (72.0 / 1200.0)
	rightsideHeight := bodyHeight + extra
	padding := 2 * thickness // Hbox(2*thickness) on each side of the body

	// vlist_out stacks (Hrule total = thickness) + (Glue('fill')) + (body).
	// The natural height above the body baseline is thickness + body.height; the
	// glue stretches by `stretch` to fill rightsideHeight, and vlist_out ROUNDS
	// the stretched glue — so the body baseline lands round(stretch)-stretch
	// below the sqrt baseline (a sub-pixel shift that the int-blit needs exact).
	stretch := rightsideHeight - thickness - radicandBox.Ascent
	bodyDy := math.Round(stretch) - stretch

	// Vinculum (Hrule): vlist_out advances cur_v by the Hrule's full height
	// (thickness) from the top edge BEFORE drawing, so the rule's top sits one
	// thickness below the box top, i.e. at -(rightsideHeight) + thickness.
	ruleTop := -rightsideHeight + thickness

	var out mathLayoutBox
	out.appendTranslated(root, 0, 0)
	bodyX := root.Width + padding
	out.appendTranslated(shiftMathBoxDown(radicandBox, bodyDy), bodyX, 0)
	out.Width = root.Width + radicandBox.Width + 2*padding
	// sqrt hlist hpack: height = rightside.height (= rightsideHeight), depth =
	// max(check.depth+shift, rightside.depth) = root.Descent (= radicand depth).
	out.Ascent = rightsideHeight
	out.Descent = maxFloat64(root.Descent, radicandBox.Descent)
	out.rules = append(out.rules, MathTextLayoutRule{
		Rect: Rect{
			Min: Pt{X: root.Width, Y: ruleTop},
			Max: Pt{X: out.Width, Y: ruleTop + thickness},
		},
	})

	if index != nil {
		// matplotlib sqrt: the root index is shrunk twice (SHRINK_FACTOR^2) and
		// laid out as Hlist([root_vlist, Kern(-check.width*0.5), check, rightside])
		// with the index (root_vlist) pinned at x=0. The radical therefore sits at
		// x = index.Width - check.Width*0.5 (the box origin is the index's left
		// edge). When that overhang is positive, shift the radical+body+rule right
		// by it and keep the index at 0; otherwise the radical stays at 0 and the
		// index sits root.Width*0.5 - index.Width to its left (origin at radical).
		// The index is shifted up by height*0.6 (matplotlib's hard-coded 0.6 hack).
		indexBox := layoutMathNode(r, *index, size*mathFracShrink*mathFracShrink, fontKey, opts)
		over := indexBox.Width - root.Width*0.5
		indexX := 0.0
		if over > 0 {
			out = shiftMathBoxRight(out, over)
			out.Width += over
		} else {
			indexX = -over // = root.Width*0.5 - indexBox.Width
		}
		y := -root.Ascent * 0.6
		out.appendTranslated(indexBox, indexX, y)
		out.Ascent = maxFloat64(out.Ascent, -y+indexBox.Ascent)
	}
	return out
}

func matrixEnvironmentDelimiters(name string) (left, right string, ok bool) {
	switch name {
	case "matrix", "array":
		return "", "", true
	case "pmatrix":
		return "(", ")", true
	case "bmatrix":
		return "[", "]", true
	case "Bmatrix":
		return "{", "}", true
	case "vmatrix":
		return "|", "|", true
	case "Vmatrix":
		return "‖", "‖", true
	default:
		return "", "", false
	}
}
