package mathtext

import "math"

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
