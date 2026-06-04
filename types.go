package mathtext

// Metrics is the renderer-neutral text measurement subset needed for MathText
// layout. BoundsY/BoundsH describe the signed ink bounds relative to the
// baseline when the renderer can provide them.
type Metrics struct{ W, H, Ascent, Descent, BoundsY, BoundsH float64 }

// GlyphInfo carries matplotlib `_get_info` metrics for one glyph (device pixels,
// baseline-relative, y-up). Advance is the UNHINTED linearHoriAdvance, Iceberg =
// horiBearingY/64 (TeX height), Height the full ink height, Xmin/Xmax/Ymin/Ymax
// the ink bbox, KernToPrev the kerning to the previous glyph in the run.
type GlyphInfo struct {
	Advance    float64
	Iceberg    float64
	Height     float64
	Xmin       float64
	Xmax       float64
	Ymin       float64
	Ymax       float64
	KernToPrev float64
}

// Measurer measures text for one font key and size.
type Measurer interface {
	MeasureText(text string, size float64, fontKey string) Metrics
}

// DPIMeasurer is an optional Measurer capability exposing the render DPI, so the
// layout can compute matplotlib's exact `fontsize*dpi/72` device pixel size for
// the (font-independent) underline thickness instead of reconstructing it from
// the x-height.
type DPIMeasurer interface {
	DPI() float64
}

// GlyphMeasurer is an optional Measurer capability returning matplotlib's exact
// per-glyph `_get_info` metrics. When a Measurer implements it and GlyphRun
// returns ok, the layout positions glyphs pixel-exactly; otherwise it falls back
// to whole-run MeasureText. (Unavailable on purego/WASM, which lacks FreeType.)
type GlyphMeasurer interface {
	GlyphRun(text string, size float64, fontKey string) ([]GlyphInfo, bool)
}

// FontStyle describes the font posture requested by MathText style commands.
type FontStyle string

const (
	FontStyleNormal FontStyle = "normal"
	FontStyleItalic FontStyle = "italic"
)

// FontRequest describes a MathText font override relative to the current font.
type FontRequest struct {
	Families []string
	Style    FontStyle
	Weight   int
}

// FontResolver resolves MathText font requests into renderer font keys.
type FontResolver interface {
	ResolveMathFontKey(base string, request FontRequest) string
}

// Options configures MathText layout.
type Options struct {
	FontResolver   FontResolver
	Cache          *Cache
	MeasurementKey string
}

// MathTextLayoutRun is one text draw in a laid-out MathText expression.
type MathTextLayoutRun struct {
	Text     string
	Offset   Pt
	FontSize float64
	FontKey  string
}

// MathTextLayoutRule is a filled rule, such as a fraction bar or root vinculum.
type MathTextLayoutRule struct {
	Rect Rect
}

// MathTextLayout is a lightweight layout tree flattened into draw runs and
// rules. Offsets and rectangles are relative to the expression baseline.
type MathTextLayout struct {
	Runs    []MathTextLayoutRun
	Rules   []MathTextLayoutRule
	Width   float64
	Ascent  float64
	Descent float64
	Height  float64
}
