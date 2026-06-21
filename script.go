package mathtext

import "math"

func layoutMathScript(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	if isMathLimitOperator(pointerNode(n.base)) {
		return layoutMathLimits(r, n, size, fontKey, opts)
	}

	// Faithful port of matplotlib 3.8.4 _mathtext.Parser.subsuper (regular
	// sub/superscript branch). Vertical shifts come from the DejaVu Sans font
	// constants scaled by x-height; the script box is shrunk once. Layout space
	// is y-down (negative Y above baseline); matplotlib height=ascent, depth=descent.
	base := layoutMathNode(r, pointerNode(n.base), size, fontKey, opts)
	// matplotlib wraps each script as Hlist([Kern, script]).shrink(), which
	// LINEARLY scales the base-size box metrics by SHRINK_FACTOR (it does not
	// re-measure at the smaller size — see shrinkMathBox / layoutMathFrac). So
	// lay each script out at the base size and shrink the box.
	layoutScript := func(node mathLayoutNode) mathLayoutBox {
		return shrinkMathBox(layoutMathNode(r, node, size, fontKey, opts), mathFracShrink)
	}
	xHeight := mathXHeight(r, size, fontKey)
	ruleThickness := mathUnderlineThickness(r, size, fontKey)

	baseText := nodePlainText(pointerNode(n.base))
	dropSub := isMathDropSubGlyph(baseText)
	// matplotlib is_slanted: italic variable faces are slanted, and so are the
	// drop-sub integral operators (∫∮), which render from a slanted math face.
	// The slant comes from the node's font style (FontStyleItalic), not the
	// resolved fontKey, which is empty at layout time.
	slanted := nodeIsSlanted(pointerNode(n.base)) || dropSub
	lcHeight := base.Ascent
	lcBaseline := 0.0
	if dropSub {
		lcBaseline = base.Descent
	}

	// Horizontal kerning of the scripts relative to the nucleus advance.
	superKern := mathScriptDelta * xHeight
	subKern := mathScriptDelta * xHeight
	if slanted {
		superKern += mathScriptDelta * xHeight
		superKern += mathScriptDeltaSlanted * (lcHeight - xHeight*2.0/3.0)
		if dropSub {
			subKern = (3*mathScriptDelta - mathScriptDeltaIntegral) * lcHeight
			superKern = (3*mathScriptDelta + mathScriptDeltaIntegral) * lcHeight
		} else {
			subKern = 0
		}
	}

	// matplotlib builds each script as Hlist([Kern(kern), script]) and calls
	// x.shrink() once, which scales the kern by SHRINK_FACTOR (the script box is
	// also laid out at scriptSize). The vertical shift_amount and the trailing
	// script_space are applied AFTER shrink, so they are not scaled.
	superKern *= mathFracShrink
	subKern *= mathFracShrink

	var out mathLayoutBox
	out.appendTranslated(base, 0, 0)
	out.Width = base.Width
	out.Ascent = base.Ascent
	out.Descent = base.Descent
	scriptMaxX := base.Width

	switch {
	case n.super == nil && n.sub != nil:
		// node757: subscript without superscript.
		sub := layoutScript(*n.sub)
		shiftDown := mathScriptSub1 * xHeight
		if dropSub {
			shiftDown = lcBaseline + mathScriptSubdrop*xHeight
		}
		scriptX := base.Width + subKern
		out.appendTranslated(sub, scriptX, shiftDown)
		scriptMaxX = maxFloat64(scriptMaxX, scriptX+sub.Width)
		out.Descent = maxFloat64(out.Descent, shiftDown+sub.Descent)
		out.Ascent = maxFloat64(out.Ascent, sub.Ascent-shiftDown)
	case n.super != nil:
		super := layoutScript(*n.super)
		shiftUp := mathScriptSup1 * xHeight
		if dropSub {
			shiftUp = lcHeight - mathScriptSubdrop*xHeight
		}
		superX := base.Width + superKern
		if n.sub == nil {
			out.appendTranslated(super, superX, -shiftUp)
			scriptMaxX = maxFloat64(scriptMaxX, superX+super.Width)
			out.Ascent = maxFloat64(out.Ascent, shiftUp+super.Ascent)
			out.Descent = maxFloat64(out.Descent, super.Descent-shiftUp)
		} else {
			// node759: both sub and superscript; if they would collide, raise super.
			sub := layoutScript(*n.sub)
			shiftDown := mathScriptSub2 * xHeight
			if dropSub {
				shiftDown = lcBaseline + mathScriptSubdrop*xHeight
			}
			clr := 2.0*ruleThickness - ((shiftUp - super.Descent) - (sub.Ascent - shiftDown))
			if clr > 0 {
				shiftUp += clr
			}
			subX := base.Width + subKern
			out.appendTranslated(super, superX, -shiftUp)
			out.appendTranslated(sub, subX, shiftDown)
			scriptMaxX = maxFloat64(scriptMaxX, maxFloat64(superX+super.Width, subX+sub.Width))
			out.Ascent = maxFloat64(out.Ascent, shiftUp+super.Ascent)
			out.Descent = maxFloat64(out.Descent, shiftDown+sub.Descent)
		}
	}

	// script_space trailing kern, except after drop-sub operators.
	if !dropSub {
		scriptMaxX += mathScriptSpace * xHeight
	}
	out.Width = scriptMaxX
	return out
}

// nodeIsSlanted reports whether the nucleus renders with a slanted (italic)
// face, matching matplotlib's Char.is_slanted() for the sub/superscript kerning
// adjustments. The slant is carried by the layout node's FontStyleItalic (math
// variables are implicitly italic); the resolved fontKey is empty at layout time
// so it cannot be inspected. For a multi-element nucleus the last element's slant
// is used (matplotlib keys off last_char).
func nodeIsSlanted(n mathLayoutNode) bool {
	switch n.kind {
	case mathLayoutStyled:
		if n.style == FontStyleItalic {
			return true
		}
		if n.style == FontStyleNormal && n.child != nil {
			return nodeIsSlanted(*n.child)
		}
		return false
	case mathLayoutList:
		if len(n.children) == 0 {
			return false
		}
		return nodeIsSlanted(n.children[len(n.children)-1])
	default:
		return false
	}
}

func layoutMathLimits(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	base := layoutMathNode(r, pointerNode(n.base), size, fontKey, opts)
	// As in layoutMathScript, the limits are shrunk via linear box scaling
	// (matplotlib Node.shrink()), not re-measured at the smaller font size.
	layoutScript := func(node mathLayoutNode) mathLayoutBox {
		return shrinkMathBox(layoutMathNode(r, node, size, fontKey, opts), mathFracShrink)
	}

	var super, sub mathLayoutBox
	if n.super != nil {
		super = layoutScript(*n.super)
	}
	if n.sub != nil {
		sub = layoutScript(*n.sub)
	}

	// matplotlib HCenters the nucleus by its Char.width = INK width (metrics.width),
	// not the advance — and a big operator like ∏ has large side bearings (advance
	// 14.85 vs ink 12.0), so advance-centering shifts it ~2px. Use the nucleus ink
	// width for a single display-operator glyph; multi-glyph nuclei (\lim) keep the
	// Hlist advance width (their trailing kern already folds advance into width).
	baseCenterWidth := base.Width
	baseText := nodePlainText(pointerNode(n.base))
	if isMathDisplayOperatorGlyph(baseText) {
		if gm, ok := r.(GlyphMeasurer); ok {
			opFontKey := mathDisplayOperatorFontKey(baseText, fontKey)
			if infos, ok := gm.GlyphRun(baseText, size, opFontKey); ok && len(infos) == 1 {
				baseCenterWidth = infos[0].Xmax - infos[0].Xmin
			}
		}
	}

	width := baseCenterWidth
	if super.Width > width {
		width = super.Width
	}
	if sub.Width > width {
		width = sub.Width
	}

	// matplotlib over/under limits stack HCentered rows; hlist_out rounds the
	// centering glue (see layoutMathFrac), so round each row's left pad.
	baseX := math.RoundToEven((width - baseCenterWidth) / 2)
	superX := math.RoundToEven((width - super.Width) / 2)
	subX := math.RoundToEven((width - sub.Width) / 2)
	// matplotlib 3.8.4 subsuper over/under stack: vgap = rule_thickness * 3.
	gap := mathUnderlineThickness(r, size, fontKey) * 3.0

	var out mathLayoutBox
	out.Width = width
	out.appendTranslated(base, baseX, 0)
	out.Ascent = base.Ascent
	out.Descent = base.Descent

	if n.super != nil {
		y := -(base.Ascent + gap + super.Descent)
		out.appendTranslated(super, superX, y)
		out.Ascent = maxFloat64(out.Ascent, -y+super.Ascent)
		out.Descent = maxFloat64(out.Descent, y+super.Descent)
	}
	if n.sub != nil {
		y := base.Descent + gap + sub.Ascent
		out.appendTranslated(sub, subX, y)
		out.Ascent = maxFloat64(out.Ascent, -y+sub.Ascent)
		out.Descent = maxFloat64(out.Descent, y+sub.Descent)
	}
	return out
}

func isMathLimitOperator(n mathLayoutNode) bool {
	return n.kind == mathLayoutText && isMathLimitText(n.text)
}

// mathOverUnderFunctionTexts are matplotlib's _overunder_functions (mapped to
// their display text); these do NOT receive the operatorname trailing thin space.
var mathOverUnderFunctionTexts = map[string]bool{
	"lim": true, "lim inf": true, "lim sup": true, "sup": true, "max": true, "min": true,
}

// mathFunctionTakesThinSpace reports whether a function operator gets a trailing
// \, thin space (all functions except the over-under ones).
func mathFunctionTakesThinSpace(op string) bool {
	return !mathOverUnderFunctionTexts[op]
}

var mathTextSpacedCommandSymbols = map[string]struct{}{
	"pm":             {},
	"mp":             {},
	"times":          {},
	"cdot":           {},
	"div":            {},
	"ast":            {},
	"circ":           {},
	"bullet":         {},
	"le":             {},
	"leq":            {},
	"ge":             {},
	"geq":            {},
	"ne":             {},
	"neq":            {},
	"approx":         {},
	"equiv":          {},
	"propto":         {},
	"sim":            {},
	"in":             {},
	"notin":          {},
	"subset":         {},
	"subseteq":       {},
	"supset":         {},
	"supseteq":       {},
	"cup":            {},
	"cap":            {},
	"land":           {},
	"lor":            {},
	"oplus":          {},
	"otimes":         {},
	"to":             {},
	"rightarrow":     {},
	"leftarrow":      {},
	"leftrightarrow": {},
}

func mathTextCommandNeedsOperatorSpacing(name string) bool {
	if _, ok := mathTextSpacedCommandSymbols[name]; ok {
		return true
	}
	_, ok := mathTex2UniSpacedNames[name]
	return ok
}

// mathNextCharSuppressesFunctionSpace reports whether the next non-space char
// after a function name is a delimiter or a sub/superscript, in which case
// matplotlib omits the operatorname thin space.
func mathNextCharSuppressesFunctionSpace(input []rune, pos int) bool {
	for pos < len(input) && (input[pos] == ' ' || input[pos] == '\t' || input[pos] == '\n' || input[pos] == '\r') {
		pos++
	}
	if pos >= len(input) {
		return true
	}
	switch input[pos] {
	case '^', '_', '(', ')', '[', ']', '|', '/':
		return true
	default:
		return false
	}
}

func isMathDropSubGlyph(text string) bool {
	switch text {
	case "∫", "∮":
		return true
	default:
		return false
	}
}

func isMathLimitText(text string) bool {
	switch text {
	case "∑", "∏", "lim", "lim inf", "lim sup", "max", "min", "sup", "inf":
		return true
	default:
		return false
	}
}
