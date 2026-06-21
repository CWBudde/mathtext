package mathtext

import "strings"

// Math alphabet translation maps ASCII letters/digits to the Unicode
// Mathematical Alphanumeric Symbols block (U+1D400…), mirroring matplotlib's
// `\mathbb \mathcal \mathscr \mathfrak \boldsymbol \bm` handling. Glyphs that
// the primary font lacks are resolved by the renderer's per-glyph font
// fallback (e.g. to STIXGeneral). Runes with no mapping pass through unchanged
// so nothing is dropped.

// mathAlphabetStyle describes one alphabet: contiguous base code points for
// uppercase, lowercase, and digits (0 means that class is not translated), plus
// a table of reserved-code-point holes that live in the Letterlike Symbols
// block instead of the contiguous range.
type mathAlphabetStyle struct {
	upper rune
	lower rune
	digit rune
	holes map[rune]rune
}

var mathAlphabetStyles = map[string]mathAlphabetStyle{
	// Double-struck (blackboard bold): \mathbb.
	"bb": {
		upper: 0x1D538, lower: 0x1D552, digit: 0x1D7D8,
		holes: map[rune]rune{
			'C': 0x2102, 'H': 0x210D, 'N': 0x2115, 'P': 0x2119,
			'Q': 0x211A, 'R': 0x211D, 'Z': 0x2124,
		},
	},
	// Script / calligraphic: \mathcal, \mathscr.
	"scr": {
		upper: 0x1D49C, lower: 0x1D4B6,
		holes: map[rune]rune{
			'B': 0x212C, 'E': 0x2130, 'F': 0x2131, 'H': 0x210B,
			'I': 0x2110, 'L': 0x2112, 'M': 0x2133, 'R': 0x211B,
			'e': 0x212F, 'g': 0x210A, 'o': 0x2134,
		},
	},
	// Fraktur (Gothic): \mathfrak.
	"frak": {
		upper: 0x1D504, lower: 0x1D51E,
		holes: map[rune]rune{
			'C': 0x212D, 'H': 0x210C, 'I': 0x2111, 'R': 0x211C, 'Z': 0x2128,
		},
	},
	// Bold (digits) used by \boldsymbol for non-letters.
	"bf": {upper: 0x1D400, lower: 0x1D41A, digit: 0x1D7CE},
	// Bold italic used by \boldsymbol for Latin letters.
	"bfit": {upper: 0x1D468, lower: 0x1D482},
}

// mathBoldsymbolStyles maps a command name to the per-class styles \boldsymbol
// and \bm apply: bold italic for Latin letters, bold for digits, and a
// pass-through for everything else (matplotlib uses bfit for letters, bf
// otherwise).
var mathBoldsymbolStyles = map[string]struct{}{
	"boldsymbol": {},
	"bm":         {},
}

// mathAlphabetCommandName reports whether name is a recognized alphabet command.
func mathAlphabetCommandName(name string) bool {
	switch name {
	case "mathbb", "mathcal", "mathscr", "mathfrak":
		return true
	}
	_, ok := mathBoldsymbolStyles[name]
	return ok
}

// mathAlphabetCommand reports whether name is a recognized alphabet command and,
// if so, returns the translated form of arg.
func mathAlphabetCommand(name, arg string) (string, bool) {
	switch name {
	case "mathbb":
		return translateMathAlphabet("bb", arg), true
	case "mathcal", "mathscr":
		return translateMathAlphabet("scr", arg), true
	case "mathfrak":
		return translateMathAlphabet("frak", arg), true
	}
	if _, ok := mathBoldsymbolStyles[name]; ok {
		return translateBoldsymbol(arg), true
	}
	return "", false
}

// translateMathAlphabet maps each rune of s through the named alphabet style.
func translateMathAlphabet(style, s string) string {
	st, ok := mathAlphabetStyles[style]
	if !ok {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		b.WriteRune(mapMathAlphabetRune(st, r))
	}
	return b.String()
}

// translateBoldsymbol applies \boldsymbol semantics: Latin letters become bold
// italic, digits become bold, all other runes are left untouched.
func translateBoldsymbol(s string) string {
	letters := mathAlphabetStyles["bfit"]
	digits := mathAlphabetStyles["bf"]
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z':
			b.WriteRune(mapMathAlphabetRune(letters, r))
		case r >= '0' && r <= '9':
			b.WriteRune(mapMathAlphabetRune(digits, r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func mapMathAlphabetRune(st mathAlphabetStyle, r rune) rune {
	if h, ok := st.holes[r]; ok {
		return h
	}
	switch {
	case r >= 'A' && r <= 'Z' && st.upper != 0:
		return st.upper + (r - 'A')
	case r >= 'a' && r <= 'z' && st.lower != 0:
		return st.lower + (r - 'a')
	case r >= '0' && r <= '9' && st.digit != 0:
		return st.digit + (r - '0')
	default:
		return r
	}
}
