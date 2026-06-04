package mathtext

import (
	"strings"
	"unicode"
)

type mathLayoutKind uint8

const (
	mathLayoutList mathLayoutKind = iota
	mathLayoutText
	mathLayoutScript
	mathLayoutFrac
	mathLayoutSqrt
	mathLayoutSpace
	mathLayoutStyled
	mathLayoutFence
	mathLayoutMatrix
)

const (
	mathGlyphDelimiterScale = 0.45
	mathRuleDelimiterScale  = 0.76
	mathSpaceScale          = 1.60
	mathCMXHeightScale      = 0.43
)

type mathDelimiterGlyph struct {
	text    string
	fontKey string
}

type mathLayoutNode struct {
	kind     mathLayoutKind
	text     string
	widthEm  float64
	children []mathLayoutNode
	segments []mathLayoutNode
	base     *mathLayoutNode
	super    *mathLayoutNode
	sub      *mathLayoutNode
	num      *mathLayoutNode
	den      *mathLayoutNode
	radicand *mathLayoutNode
	index    *mathLayoutNode
	child    *mathLayoutNode
	left     string
	middles  []string
	right    string
	fracRule bool
	fracDisp bool
	rows     [][]mathLayoutNode
	families []string
	style    FontStyle
	weight   int
	spaced   bool
}

type mathLayoutBox struct {
	runs    []MathTextLayoutRun
	rules   []MathTextLayoutRule
	Width   float64
	Ascent  float64
	Descent float64
}

func appendMathText(children []mathLayoutNode, text string) []mathLayoutNode {
	if text == "" {
		return children
	}
	n := len(children)
	if n > 0 && children[n-1].kind == mathLayoutText {
		children[n-1].text += text
		return children
	}
	return append(children, mathLayoutNode{kind: mathLayoutText, text: text})
}

func appendMathAtom(children []mathLayoutNode, r rune, implicitItalic bool) []mathLayoutNode {
	node := mathAtomNode(r, implicitItalic)
	if node.kind == mathLayoutText {
		return appendMathText(children, node.text)
	}
	return append(children, node)
}

func mathAtomNode(r rune, implicitItalic bool) mathLayoutNode {
	if implicitItalic && isMathItalicRune(r) {
		text := mathLayoutNode{kind: mathLayoutText, text: string(r)}
		return mathLayoutNode{kind: mathLayoutStyled, child: &text, style: FontStyleItalic}
	}
	return mathLayoutNode{kind: mathLayoutText, text: string(r)}
}

func isMathItalicRune(r rune) bool {
	if !unicode.IsLetter(r) {
		return false
	}
	return !(unicode.In(r, unicode.Greek) && unicode.IsUpper(r))
}

func attachMathScript(children []mathLayoutNode, marker rune, script mathLayoutNode) []mathLayoutNode {
	if len(children) == 0 {
		return append(children, mathLayoutNode{kind: mathLayoutText, text: string(marker) + nodePlainText(script)})
	}
	last := children[len(children)-1]
	if last.kind != mathLayoutScript {
		base := last
		last = mathLayoutNode{kind: mathLayoutScript, base: &base}
	}
	if marker == '^' {
		last.super = &script
	} else {
		last.sub = &script
	}
	children[len(children)-1] = last
	return children
}

func scriptTargetsOverUnderFunction(children []mathLayoutNode) bool {
	if len(children) == 0 {
		return false
	}
	last := children[len(children)-1]
	if last.kind == mathLayoutScript {
		last = pointerNode(last.base)
	}
	return last.kind == mathLayoutText && mathOverUnderFunctionTexts[last.text]
}

func (n mathLayoutNode) isEmpty() bool {
	return n.kind == mathLayoutText && n.text == ""
}

func nodePlainText(n mathLayoutNode) string {
	switch n.kind {
	case mathLayoutText:
		return n.text
	case mathLayoutList:
		var out strings.Builder
		for _, child := range n.children {
			out.WriteString(nodePlainText(child))
		}
		return out.String()
	case mathLayoutScript:
		base := nodePlainText(pointerNode(n.base))
		if n.sub != nil {
			base += "_" + nodePlainText(*n.sub)
		}
		if n.super != nil {
			base += "^" + nodePlainText(*n.super)
		}
		return base
	case mathLayoutFrac:
		return formatMathFraction(nodePlainText(pointerNode(n.num)), nodePlainText(pointerNode(n.den)))
	case mathLayoutSqrt:
		return "√" + groupMathAtom(nodePlainText(pointerNode(n.radicand)))
	case mathLayoutStyled:
		return nodePlainText(pointerNode(n.child))
	case mathLayoutFence:
		var out strings.Builder
		if n.left != "" {
			out.WriteString(n.left)
		}
		for i, segment := range fenceSegments(n) {
			if i > 0 && i-1 < len(n.middles) {
				out.WriteString(n.middles[i-1])
			}
			out.WriteString(nodePlainText(segment))
		}
		if n.right != "" {
			out.WriteString(n.right)
		}
		return out.String()
	case mathLayoutMatrix:
		var out strings.Builder
		if n.left != "" {
			out.WriteString(n.left)
		}
		for i, row := range n.rows {
			if i > 0 {
				out.WriteString("; ")
			}
			for j, cell := range row {
				if j > 0 {
					out.WriteByte(' ')
				}
				out.WriteString(nodePlainText(cell))
			}
		}
		if n.right != "" {
			out.WriteString(n.right)
		}
		return out.String()
	default:
		return ""
	}
}

func pointerNode(n *mathLayoutNode) mathLayoutNode {
	if n == nil {
		return mathLayoutNode{kind: mathLayoutText}
	}
	return *n
}

func fenceSegments(n mathLayoutNode) []mathLayoutNode {
	if len(n.segments) > 0 {
		return n.segments
	}
	if n.child != nil {
		return []mathLayoutNode{*n.child}
	}
	return nil
}

func (b *mathLayoutBox) appendTranslated(child mathLayoutBox, dx, dy float64) {
	for _, run := range child.runs {
		run.Offset.X += dx
		run.Offset.Y += dy
		b.runs = append(b.runs, run)
	}
	for _, rule := range child.rules {
		rule.Rect.Min.X += dx
		rule.Rect.Max.X += dx
		rule.Rect.Min.Y += dy
		rule.Rect.Max.Y += dy
		b.rules = append(b.rules, rule)
	}
}

func shrinkMathBox(box mathLayoutBox, factor float64) mathLayoutBox {
	out := mathLayoutBox{
		Width:   box.Width * factor,
		Ascent:  box.Ascent * factor,
		Descent: box.Descent * factor,
	}
	for _, run := range box.runs {
		run.Offset.X *= factor
		run.Offset.Y *= factor
		run.FontSize *= factor
		out.runs = append(out.runs, run)
	}
	for _, rule := range box.rules {
		rule.Rect.Min.X *= factor
		rule.Rect.Min.Y *= factor
		rule.Rect.Max.X *= factor
		rule.Rect.Max.Y *= factor
		out.rules = append(out.rules, rule)
	}
	return out
}

// shiftMathBoxRight shifts a box's contents right by dx, preserving metrics.
func shiftMathBoxRight(box mathLayoutBox, dx float64) mathLayoutBox {
	var out mathLayoutBox
	out.appendTranslated(box, dx, 0)
	out.Width = box.Width
	out.Ascent = box.Ascent
	out.Descent = box.Descent
	return out
}

// shiftMathBoxDown shifts a box's contents down by dy (layout y-down), updating
// ascent/descent (matplotlib Char.shift_amount).
func shiftMathBoxDown(box mathLayoutBox, dy float64) mathLayoutBox {
	var out mathLayoutBox
	out.appendTranslated(box, 0, dy)
	out.Width = box.Width
	out.Ascent = box.Ascent - dy
	out.Descent = box.Descent + dy
	return out
}

func centerMathDelimiterBox(box mathLayoutBox, targetAscent, targetDescent float64) mathLayoutBox {
	if box.Width <= 0 && len(box.runs) == 0 && len(box.rules) == 0 {
		return mathLayoutBox{}
	}
	targetCenter := (targetDescent - targetAscent) / 2
	boxCenter := (box.Descent - box.Ascent) / 2
	dy := targetCenter - boxCenter
	var out mathLayoutBox
	out.appendTranslated(box, 0, dy)
	out.Width = box.Width
	out.Ascent = maxFloat64(targetAscent, -dy+box.Ascent)
	out.Descent = maxFloat64(targetDescent, dy+box.Descent)
	return out
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
