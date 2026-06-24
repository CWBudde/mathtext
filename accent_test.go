package mathtext

import (
	"math"
	"testing"
)

// accentGlyphMeasurer is a GlyphMeasurer whose metrics scale linearly with the
// font size, so wide-accent scaling and stacking offsets are exercised.
type accentGlyphMeasurer struct{}

func (accentGlyphMeasurer) MeasureText(text string, size float64, _ string) Metrics {
	n := float64(len([]rune(text)))
	return Metrics{
		W:       n * size * 0.5,
		H:       size,
		Ascent:  size * 0.8,
		Descent: size * 0.2,
		BoundsY: -size * 0.8,
		BoundsH: size,
	}
}

func (accentGlyphMeasurer) GlyphRun(text string, size float64, _ string) ([]GlyphInfo, bool) {
	runes := []rune(text)
	out := make([]GlyphInfo, 0, len(runes))
	for _, r := range runes {
		info := GlyphInfo{
			Advance: size * 0.5,
			Xmin:    size * 0.05,
			Xmax:    size * 0.45,
			Iceberg: size * 0.7,
			Height:  size * 0.7,
			Ymin:    0,
			Ymax:    size * 0.7,
		}
		if isCombiningAccentRune(r) {
			// Combining accent glyphs float above the baseline with a small ink box.
			info.Xmin = size * 0.10
			info.Xmax = size * 0.40
			info.Ymin = size * 0.45
			info.Ymax = size * 0.70
			info.Height = size * 0.25
			info.Iceberg = size * 0.70
		}
		out = append(out, info)
	}
	return out, true
}

func isCombiningAccentRune(r rune) bool {
	return (r >= 0x300 && r <= 0x36f) || (r >= 0x20d0 && r <= 0x20f0)
}

func accentRuns(t *testing.T, expr string) MathTextLayout {
	t.Helper()
	layout, ok := LayoutMathText(accentGlyphMeasurer{}, expr, 20, "base", Options{})
	if !ok {
		t.Fatalf("LayoutMathText(%q) failed", expr)
	}
	return layout
}

func findRun(runs []MathTextLayoutRun, text string) (MathTextLayoutRun, bool) {
	for _, run := range runs {
		if run.Text == text {
			return run, true
		}
	}
	return MathTextLayoutRun{}, false
}

func TestParseAccentNode(t *testing.T) {
	node := parseMathLayoutNode(`\hat{x}`, nil)
	// The layout parser wraps single children in a list.
	if node.kind != mathLayoutList || len(node.children) != 1 {
		t.Fatalf("unexpected top node: %+v", node)
	}
	accent := node.children[0]
	if accent.kind != mathLayoutAccent {
		t.Fatalf("expected mathLayoutAccent, got kind %d", accent.kind)
	}
	if accent.accent != string(rune(0x302)) {
		t.Fatalf("expected circumflex accent glyph U+0302, got %q", accent.accent)
	}
	if accent.accentWide {
		t.Fatalf("\\hat must not be a wide accent")
	}
}

func TestParseMathringRings(t *testing.T) {
	node := parseMathLayoutNode(`\mathring{a}`, nil).children[0]
	if node.kind != mathLayoutAccent || node.accentRings != 2 {
		t.Fatalf("expected mathring with 2 shrink rings, got %+v", node)
	}
}

func TestLayoutSingleAccentCentersAboveNucleus(t *testing.T) {
	plain := accentRuns(t, `x`)
	hat := accentRuns(t, `\hat{x}`)

	nucleus, ok := findRun(hat.Runs, "x")
	if !ok {
		t.Fatalf("missing nucleus run: %+v", hat.Runs)
	}
	accent, ok := findRun(hat.Runs, string(rune(0x302)))
	if !ok {
		t.Fatalf("missing accent run: %+v", hat.Runs)
	}
	if accent.FontSize != 20 {
		t.Fatalf("single accent must render at the base size, got %v", accent.FontSize)
	}
	// The accent sits above the baseline (negative Y in y-down layout space).
	if accent.Offset.Y >= 0 {
		t.Fatalf("accent should be above the baseline, got Y=%v", accent.Offset.Y)
	}
	// The accented box is taller than the bare nucleus.
	if hat.Ascent <= plain.Ascent {
		t.Fatalf("accent should increase ascent: %v <= %v", hat.Ascent, plain.Ascent)
	}
	// Width is unchanged (matplotlib packs the accent to the nucleus width).
	if math.Abs(hat.Width-plain.Width) > 1e-9 {
		t.Fatalf("accent width should equal nucleus width: %v vs %v", hat.Width, plain.Width)
	}
	_ = nucleus
}

func TestLayoutWideAccentScalesGlyph(t *testing.T) {
	wide := accentRuns(t, `\widehat{xy}`)
	accent, ok := findRun(wide.Runs, string(rune(0x302)))
	if !ok {
		t.Fatalf("missing wide accent run: %+v", wide.Runs)
	}
	// The caret is scaled up to span the two-character nucleus.
	if accent.FontSize <= 20 {
		t.Fatalf("wide accent should scale the glyph above the base size, got %v", accent.FontSize)
	}
}

func TestLayoutOverlineEmitsRule(t *testing.T) {
	layout := accentRuns(t, `\overline{x}`)
	if len(layout.Rules) != 1 {
		t.Fatalf("expected exactly one overline rule, got %d", len(layout.Rules))
	}
	rule := layout.Rules[0].Rect
	if rule.Min.Y >= 0 {
		t.Fatalf("overline rule should be above the baseline, got %+v", rule)
	}
	if rule.Max.X <= rule.Min.X {
		t.Fatalf("overline rule has non-positive width: %+v", rule)
	}
}

func TestLayoutOversetStacksAnnotation(t *testing.T) {
	layout := accentRuns(t, `\overset{a}{b}`)
	annot, ok := findRun(layout.Runs, "a")
	if !ok {
		t.Fatalf("missing annotation run: %+v", layout.Runs)
	}
	base, ok := findRun(layout.Runs, "b")
	if !ok {
		t.Fatalf("missing base run: %+v", layout.Runs)
	}
	if annot.FontSize >= base.FontSize {
		t.Fatalf("overset annotation should be shrunk: annot=%v base=%v", annot.FontSize, base.FontSize)
	}
	if annot.Offset.Y >= base.Offset.Y {
		t.Fatalf("overset annotation should sit above the base: annot Y=%v base Y=%v", annot.Offset.Y, base.Offset.Y)
	}
}

func TestLayoutUndersetStacksBelow(t *testing.T) {
	layout := accentRuns(t, `\underset{a}{b}`)
	annot, _ := findRun(layout.Runs, "a")
	base, _ := findRun(layout.Runs, "b")
	if annot.Offset.Y <= base.Offset.Y {
		t.Fatalf("underset annotation should sit below the base: annot Y=%v base Y=%v", annot.Offset.Y, base.Offset.Y)
	}
}

func TestLayoutSubstackStacksLines(t *testing.T) {
	layout := accentRuns(t, `\substack{a \\ b}`)
	a, ok := findRun(layout.Runs, "a")
	if !ok {
		t.Fatalf("missing first line: %+v", layout.Runs)
	}
	b, ok := findRun(layout.Runs, "b")
	if !ok {
		t.Fatalf("missing second line: %+v", layout.Runs)
	}
	// The first line sits above the second (more negative Y).
	if a.Offset.Y >= b.Offset.Y {
		t.Fatalf("substack lines not stacked: a Y=%v b Y=%v", a.Offset.Y, b.Offset.Y)
	}
}

func TestLayoutNotOverlay(t *testing.T) {
	layout := accentRuns(t, `\not=`)
	// \not overlays a combining long solidus (U+0338) on the following atom; the
	// layout emits it as its own glyph run (combining marks carry ~0 advance in
	// real fonts, so they overlay the base).
	if _, ok := findRun(layout.Runs, "="); !ok {
		t.Fatalf("expected base '=' run for \\not=, got %+v", layout.Runs)
	}
	if _, ok := findRun(layout.Runs, string(rune(0x338))); !ok {
		t.Fatalf("expected combining-solidus overlay run for \\not=, got %+v", layout.Runs)
	}
}

func TestCollapseEscapedBackslashesPreservesSeparator(t *testing.T) {
	// \\command collapses; a standalone \\ separator survives.
	if got := collapseEscapedBackslashes(`\\frac`); got != `\frac` {
		t.Fatalf("expected \\frac, got %q", got)
	}
	if got := collapseEscapedBackslashes(`a \\ b`); got != `a \\ b` {
		t.Fatalf("expected separator preserved, got %q", got)
	}
}

func TestNormalizeAccentFallback(t *testing.T) {
	// Plain-text fallback still applies a combining mark.
	if got := Normalize(`\breve{a}`); got != "a"+string(rune(0x306)) {
		t.Fatalf("expected combining breve fallback, got %q", got)
	}
	if got := Normalize(`\overset{!}{=}`); got != "! =" {
		t.Fatalf("expected overset fallback, got %q", got)
	}
}
