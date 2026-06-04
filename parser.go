package mathtext

import (
	"strings"
	"unicode"
)

type mathLayoutParser struct {
	input                   []rune
	pos                     int
	implicitItalic          bool
	suppressOperatorSpacing bool
}

func parseMathLayoutNode(expr string, cache *Cache) mathLayoutNode {
	if cache != nil {
		if node, ok := cache.parsedNode(expr); ok {
			return node
		}
	}
	parser := mathLayoutParser{input: []rune(expr), implicitItalic: true}
	node := parser.parseUntil(0)
	if cache != nil {
		cache.storeParsedNode(expr, node)
	}
	return node
}

func (p *mathLayoutParser) parseUntil(stop rune) mathLayoutNode {
	var children []mathLayoutNode
	appendText := func(text string) {
		children = appendMathText(children, text)
	}

	for p.pos < len(p.input) {
		r := p.input[p.pos]
		if stop != 0 && r == stop {
			break
		}
		switch r {
		case '{':
			p.pos++
			children = append(children, p.parseUntil('}').children...)
			if p.pos < len(p.input) && p.input[p.pos] == '}' {
				p.pos++
			}
		case '}':
			if stop == 0 {
				p.pos++
				continue
			}
			return mathLayoutNode{kind: mathLayoutList, children: children}
		case '^', '_':
			p.pos++
			children = attachMathScript(children, r, p.parseScriptArgumentNode(scriptTargetsOverUnderFunction(children)))
		case '\\':
			node := p.parseCommandNode()
			if node.spaced {
				children = p.appendMathSpacedOperator(children, node.text, stop)
			} else if node.kind == mathLayoutText {
				appendText(node.text)
			} else if !node.isEmpty() {
				children = append(children, node)
			}
		case '~':
			appendText(" ")
			p.pos++
		case '+', '-', '=':
			children = p.appendMathOperator(children, r, stop)
		case ',', ';', '.', '!':
			children = p.appendMathPunctuation(children, r, stop)
		case ' ', '\t', '\n', '\r':
			if !p.implicitItalic {
				appendText(" ")
			}
			p.pos++
		default:
			children = appendMathAtom(children, r, p.implicitItalic)
			p.pos++
		}
	}
	return mathLayoutNode{kind: mathLayoutList, children: children}
}

func (p *mathLayoutParser) parseArgumentNode() mathLayoutNode {
	p.skipSpace()
	if p.pos >= len(p.input) {
		return mathLayoutNode{kind: mathLayoutText}
	}
	switch p.input[p.pos] {
	case '{':
		p.pos++
		node := p.parseUntil('}')
		if p.pos < len(p.input) && p.input[p.pos] == '}' {
			p.pos++
		}
		return node
	case '\\':
		return p.parseCommandNode()
	default:
		r := p.input[p.pos]
		p.pos++
		return mathAtomNode(r, p.implicitItalic)
	}
}

func (p *mathLayoutParser) parseScriptArgumentNode(suppressOperatorSpacing bool) mathLayoutNode {
	if !suppressOperatorSpacing {
		return p.parseArgumentNode()
	}
	old := p.suppressOperatorSpacing
	p.suppressOperatorSpacing = true
	node := p.parseArgumentNode()
	p.suppressOperatorSpacing = old
	return node
}

func (p *mathLayoutParser) parseCommandNode() mathLayoutNode {
	p.pos++
	if p.pos >= len(p.input) {
		return mathLayoutNode{kind: mathLayoutText, text: `\`}
	}
	r := p.input[p.pos]
	if !unicode.IsLetter(r) {
		p.pos++
		switch r {
		case ',':
			return mathLayoutNode{kind: mathLayoutSpace, widthEm: 0.166}
		case ':':
			return mathLayoutNode{kind: mathLayoutSpace, widthEm: 0.222}
		case ';':
			return mathLayoutNode{kind: mathLayoutSpace, widthEm: 0.278}
		case ' ':
			return mathLayoutNode{kind: mathLayoutSpace, widthEm: 0.333}
		case '!':
			return mathLayoutNode{kind: mathLayoutSpace, widthEm: -0.166}
		default:
			return mathLayoutNode{kind: mathLayoutText, text: string(r)}
		}
	}

	start := p.pos
	for p.pos < len(p.input) && unicode.IsLetter(p.input[p.pos]) {
		p.pos++
	}
	name := string(p.input[start:p.pos])

	if mapped, ok := mathTextCommandMap[name]; ok {
		if mathTextCommandNeedsOperatorSpacing(name) {
			return mathLayoutNode{kind: mathLayoutText, text: mapped, spaced: true}
		}
		if runes := []rune(mapped); len(runes) == 1 {
			return mathAtomNode(runes[0], p.implicitItalic)
		}
		return mathLayoutNode{kind: mathLayoutText, text: mapped}
	}
	if width, ok := mathTextSpacingCommandWidths[name]; ok {
		return mathLayoutNode{kind: mathLayoutSpace, widthEm: width}
	}
	if delim, ok := mathTextDelimiterCommands[name]; ok {
		return mathLayoutNode{kind: mathLayoutText, text: delim}
	}
	if op, ok := mathTextOperatorMap[name]; ok {
		// matplotlib operatorname(): a function name gets a trailing thin space
		// (\, = 0.16667em) unless it is an over-under function (lim/sup/max/...) or
		// is immediately followed by a delimiter or a sub/superscript.
		if mathFunctionTakesThinSpace(op) && !mathNextCharSuppressesFunctionSpace(p.input, p.pos) {
			return mathLayoutNode{kind: mathLayoutList, children: []mathLayoutNode{
				{kind: mathLayoutText, text: op},
				{kind: mathLayoutSpace, widthEm: 0.16667},
			}}
		}
		return mathLayoutNode{kind: mathLayoutText, text: op}
	}
	if _, ok := mathTextPassthroughCommands[name]; ok {
		return p.parseStyledArgumentNode(name)
	}
	if mark, ok := mathTextAccentMarks[name]; ok {
		return mathLayoutNode{kind: mathLayoutText, text: applyMathAccent(nodePlainText(p.parseArgumentNode()), mark)}
	}
	if name == "begin" {
		return p.parseEnvironmentNode()
	}
	if name == "left" {
		return p.parseFenceNode()
	}
	if _, ok := mathTextEmptyCommands[name]; ok {
		p.skipSpace()
		return mathLayoutNode{kind: mathLayoutText}
	}

	switch name {
	case "frac":
		num := p.parseArgumentNode()
		den := p.parseArgumentNode()
		return mathLayoutNode{kind: mathLayoutFrac, num: &num, den: &den, fracRule: true}
	case "dfrac":
		num := p.parseArgumentNode()
		den := p.parseArgumentNode()
		return mathLayoutNode{kind: mathLayoutFrac, num: &num, den: &den, fracRule: true, fracDisp: true}
	case "binom":
		num := p.parseArgumentNode()
		den := p.parseArgumentNode()
		return mathLayoutNode{kind: mathLayoutFrac, num: &num, den: &den, left: "(", right: ")"}
	case "genfrac":
		left := parseMathDelimiterSpec(p.parseBraceText())
		right := parseMathDelimiterSpec(p.parseBraceText())
		rule := parseMathSpaceDimension(p.parseBraceText()) > 0
		display := strings.TrimSpace(p.parseBraceText()) == "0"
		num := p.parseArgumentNode()
		den := p.parseArgumentNode()
		return mathLayoutNode{kind: mathLayoutFrac, num: &num, den: &den, left: left, right: right, fracRule: rule, fracDisp: display}
	case "hspace", "kern":
		return mathLayoutNode{kind: mathLayoutSpace, widthEm: parseMathSpaceDimension(p.parseBraceText())}
	case "sqrt":
		var index *mathLayoutNode
		if p.pos < len(p.input) && p.input[p.pos] == '[' {
			p.pos++
			idx := p.parseUntil(']')
			if p.pos < len(p.input) && p.input[p.pos] == ']' {
				p.pos++
			}
			index = &idx
		}
		radicand := p.parseArgumentNode()
		return mathLayoutNode{kind: mathLayoutSqrt, radicand: &radicand, index: index}
	default:
		return mathLayoutNode{kind: mathLayoutText, text: `\` + name}
	}
}

func (p *mathLayoutParser) parseEnvironmentNode() mathLayoutNode {
	name := p.parseBraceText()
	left, right, ok := matrixEnvironmentDelimiters(name)
	if !ok {
		return mathLayoutNode{kind: mathLayoutText}
	}
	if name == "array" && p.pos < len(p.input) && p.input[p.pos] == '{' {
		_ = p.parseBraceText()
	}
	rows := p.parseMatrixRows(name)
	return mathLayoutNode{kind: mathLayoutMatrix, rows: rows, left: left, right: right}
}

func (p *mathLayoutParser) parseStyledArgumentNode(name string) mathLayoutNode {
	implicitItalic := p.implicitItalic
	p.implicitItalic = false
	arg := p.parseArgumentNode()
	p.implicitItalic = implicitItalic
	node := mathLayoutNode{kind: mathLayoutStyled, child: &arg, style: FontStyleNormal, weight: 400}
	switch name {
	case "mathsf":
		node.families = []string{"DejaVu Sans", "sans-serif"}
	case "mathtt":
		node.families = []string{"DejaVu Sans Mono", "monospace"}
	case "mathrm", "text":
		node.families = []string{"DejaVu Serif", "serif"}
	case "mathit":
		node.style = FontStyleItalic
	case "mathbf":
		node.weight = 700
	case "operatorname":
		// Preserve current face selection but normalize posture/weight.
	default:
		return arg
	}
	return node
}

func mathDelimiterFontKey() string {
	return "DejaVu Serif"
}

func (p *mathLayoutParser) appendMathOperator(children []mathLayoutNode, op rune, stop rune) []mathLayoutNode {
	text := string(op)
	if op == '-' {
		text = "−"
	}
	children = p.appendMathSpacedOperator(children, text, stop)
	p.pos++
	return children
}

func (p *mathLayoutParser) appendMathSpacedOperator(children []mathLayoutNode, text string, stop rune) []mathLayoutNode {
	if p.suppressOperatorSpacing {
		children = appendMathText(children, text)
	} else if p.hasPreviousMathOperand(children) && p.hasNextMathOperand(stop) {
		children = append(children, mathLayoutNode{kind: mathLayoutSpace, widthEm: 0.2})
		children = appendMathText(children, text)
		children = append(children, mathLayoutNode{kind: mathLayoutSpace, widthEm: 0.2})
	} else {
		children = appendMathText(children, text)
	}
	return children
}

func (p *mathLayoutParser) appendMathPunctuation(children []mathLayoutNode, punct rune, stop rune) []mathLayoutNode {
	text := string(punct)
	if punct == '.' && p.previousNonSpaceIsDigit() && p.nextNonSpaceIsDigit(stop) {
		children = appendMathText(children, text)
	} else {
		children = appendMathText(children, text)
		children = append(children, mathLayoutNode{kind: mathLayoutSpace, widthEm: 0.2})
	}
	p.pos++
	return children
}

func (p *mathLayoutParser) previousNonSpaceIsDigit() bool {
	for i := p.pos - 1; i >= 0; i-- {
		r := p.input[i]
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		return unicode.IsDigit(r)
	}
	return false
}

func (p *mathLayoutParser) nextNonSpaceIsDigit(stop rune) bool {
	for i := p.pos + 1; i < len(p.input); i++ {
		r := p.input[i]
		if stop != 0 && r == stop {
			return false
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		return unicode.IsDigit(r)
	}
	return false
}

func (p *mathLayoutParser) hasPreviousMathOperand(children []mathLayoutNode) bool {
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		switch child.kind {
		case mathLayoutSpace:
			continue
		case mathLayoutText:
			return child.text != "" && child.text != "+" && child.text != "−" && child.text != "="
		default:
			return true
		}
	}
	return false
}

func (p *mathLayoutParser) hasNextMathOperand(stop rune) bool {
	for i := p.pos + 1; i < len(p.input); i++ {
		r := p.input[i]
		if stop != 0 && r == stop {
			return false
		}
		switch r {
		case ' ', '\t', '\n', '\r':
			continue
		case '+', '-', '=', '^', '_', '}':
			return false
		default:
			return true
		}
	}
	return false
}

func (p *mathLayoutParser) parseFenceNode() mathLayoutNode {
	left := p.parseDelimiterToken()
	segments := []mathLayoutNode{p.parseUntilFenceBoundary()}
	middles := []string{}
	for p.consumeNamedCommand("middle") {
		middles = append(middles, p.parseDelimiterToken())
		segments = append(segments, p.parseUntilFenceBoundary())
	}
	right := ""
	if p.consumeNamedCommand("right") {
		right = p.parseDelimiterToken()
	}
	node := mathLayoutNode{kind: mathLayoutFence, left: left, middles: middles, right: right, segments: segments}
	if len(segments) == 1 {
		node.child = &segments[0]
	}
	return node
}

func (p *mathLayoutParser) parseMatrixRows(envName string) [][]mathLayoutNode {
	rows := [][]mathLayoutNode{}
	for {
		if p.startsEnvironmentEnd(envName) {
			p.consumeEnvironmentEnd(envName)
			break
		}
		row := []mathLayoutNode{}
		for {
			cell := p.parseMatrixCell(envName)
			row = append(row, cell)
			if p.startsEnvironmentEnd(envName) {
				p.consumeEnvironmentEnd(envName)
				rows = append(rows, row)
				return rows
			}
			if p.consumeMatrixRowSeparator() {
				rows = append(rows, row)
				break
			}
			if p.pos < len(p.input) && p.input[p.pos] == '&' {
				p.pos++
				continue
			}
			rows = append(rows, row)
			return rows
		}
	}
	return rows
}

func (p *mathLayoutParser) parseMatrixCell(envName string) mathLayoutNode {
	var children []mathLayoutNode
	appendText := func(text string) {
		if text == "" {
			return
		}
		n := len(children)
		if n > 0 && children[n-1].kind == mathLayoutText {
			children[n-1].text += text
			return
		}
		children = append(children, mathLayoutNode{kind: mathLayoutText, text: text})
	}

	for p.pos < len(p.input) {
		if p.startsEnvironmentEnd(envName) || p.pos < len(p.input) && p.input[p.pos] == '&' || p.startsMatrixRowSeparator() {
			break
		}
		r := p.input[p.pos]
		switch r {
		case '{':
			p.pos++
			children = append(children, p.parseUntil('}').children...)
			if p.pos < len(p.input) && p.input[p.pos] == '}' {
				p.pos++
			}
		case '^', '_':
			p.pos++
			children = attachMathScript(children, r, p.parseArgumentNode())
		case '\\':
			node := p.parseCommandNode()
			if node.kind == mathLayoutText {
				appendText(node.text)
			} else if !node.isEmpty() {
				children = append(children, node)
			}
		case '~':
			children = append(children, mathLayoutNode{kind: mathLayoutSpace, widthEm: 0.333})
			p.pos++
		case ' ', '\t', '\n', '\r':
			if !p.implicitItalic {
				appendText(" ")
			}
			p.pos++
		default:
			children = appendMathAtom(children, r, p.implicitItalic)
			p.pos++
		}
	}

	return mathLayoutNode{kind: mathLayoutList, children: children}
}

func (p *mathLayoutParser) parseUntilFenceBoundary() mathLayoutNode {
	var children []mathLayoutNode
	appendText := func(text string) {
		if text == "" {
			return
		}
		n := len(children)
		if n > 0 && children[n-1].kind == mathLayoutText {
			children[n-1].text += text
			return
		}
		children = append(children, mathLayoutNode{kind: mathLayoutText, text: text})
	}

	for p.pos < len(p.input) {
		if p.startsNamedCommand("middle") || p.startsNamedCommand("right") {
			break
		}
		r := p.input[p.pos]
		switch r {
		case '{':
			p.pos++
			children = append(children, p.parseUntil('}').children...)
			if p.pos < len(p.input) && p.input[p.pos] == '}' {
				p.pos++
			}
		case '^', '_':
			p.pos++
			children = attachMathScript(children, r, p.parseArgumentNode())
		case '\\':
			node := p.parseCommandNode()
			if node.kind == mathLayoutText {
				appendText(node.text)
			} else if !node.isEmpty() {
				children = append(children, node)
			}
		case '~':
			children = append(children, mathLayoutNode{kind: mathLayoutSpace, widthEm: 0.333})
			p.pos++
		case ' ', '\t', '\n', '\r':
			if !p.implicitItalic {
				appendText(" ")
			}
			p.pos++
		default:
			children = appendMathAtom(children, r, p.implicitItalic)
			p.pos++
		}
	}
	return mathLayoutNode{kind: mathLayoutList, children: children}
}

func (p *mathLayoutParser) parseDelimiterToken() string {
	p.skipSpace()
	if p.pos >= len(p.input) {
		return ""
	}
	if p.input[p.pos] == '.' {
		p.pos++
		return ""
	}
	if p.input[p.pos] == '\\' {
		p.pos++
		if p.pos >= len(p.input) {
			return ""
		}
		r := p.input[p.pos]
		if !unicode.IsLetter(r) {
			p.pos++
			switch r {
			case '{', '}':
				return string(r)
			case '|':
				return "|"
			default:
				return string(r)
			}
		}
		start := p.pos
		for p.pos < len(p.input) && unicode.IsLetter(p.input[p.pos]) {
			p.pos++
		}
		name := string(p.input[start:p.pos])
		if delim, ok := mathTextDelimiterCommands[name]; ok {
			return delim
		}
		if mapped, ok := mathTextCommandMap[name]; ok {
			return mapped
		}
		return ""
	}
	r := p.input[p.pos]
	p.pos++
	return string(r)
}

func parseMathDelimiterSpec(text string) string {
	parser := mathLayoutParser{input: []rune(strings.TrimSpace(text))}
	return parser.parseDelimiterToken()
}

func (p *mathLayoutParser) parseBraceText() string {
	p.skipSpace()
	if p.pos >= len(p.input) || p.input[p.pos] != '{' {
		return ""
	}
	p.pos++
	start := p.pos
	depth := 1
	for p.pos < len(p.input) {
		switch p.input[p.pos] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				text := string(p.input[start:p.pos])
				p.pos++
				return text
			}
		}
		p.pos++
	}
	return string(p.input[start:])
}

func (p *mathLayoutParser) startsNamedCommand(name string) bool {
	if p.pos >= len(p.input)-len(name) || p.input[p.pos] != '\\' {
		return false
	}
	i := p.pos + 1
	for _, want := range name {
		if i >= len(p.input) || p.input[i] != want {
			return false
		}
		i++
	}
	return i >= len(p.input) || !unicode.IsLetter(p.input[i])
}

func (p *mathLayoutParser) consumeNamedCommand(name string) bool {
	if !p.startsNamedCommand(name) {
		return false
	}
	p.pos += 1 + len([]rune(name))
	return true
}

func (p *mathLayoutParser) startsEnvironmentEnd(name string) bool {
	save := p.pos
	if !p.consumeNamedCommand("end") {
		p.pos = save
		return false
	}
	text := p.parseBraceText()
	p.pos = save
	return text == name
}

func (p *mathLayoutParser) consumeEnvironmentEnd(name string) bool {
	save := p.pos
	if !p.consumeNamedCommand("end") {
		p.pos = save
		return false
	}
	if p.parseBraceText() != name {
		p.pos = save
		return false
	}
	return true
}

func (p *mathLayoutParser) startsMatrixRowSeparator() bool {
	if p.pos+1 >= len(p.input) || p.input[p.pos] != '\\' {
		return false
	}
	if p.input[p.pos+1] == '\\' {
		return true
	}
	if !unicode.IsLetter(p.input[p.pos+1]) {
		return false
	}
	i := p.pos + 1
	for i < len(p.input) && unicode.IsLetter(p.input[i]) {
		i++
	}
	return string(p.input[p.pos+1:i]) == "cr"
}

func (p *mathLayoutParser) consumeMatrixRowSeparator() bool {
	if !p.startsMatrixRowSeparator() {
		return false
	}
	if p.input[p.pos+1] == '\\' {
		p.pos += 2
	} else {
		p.pos++
		for p.pos < len(p.input) && unicode.IsLetter(p.input[p.pos]) {
			p.pos++
		}
	}
	return true
}

func (p *mathLayoutParser) skipSpace() {
	for p.pos < len(p.input) && unicode.IsSpace(p.input[p.pos]) {
		p.pos++
	}
}
