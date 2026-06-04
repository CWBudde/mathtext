// Package mathtext parses and lays out Matplotlib-style MathText expressions
// for renderer-independent drawing.
//
// The package intentionally exposes a small renderer-neutral API surface:
//   - Normalize and NormalizeDisplay provide Unicode fallback text.
//   - SplitDisplaySegments separates plain text from inline $...$ math.
//   - LayoutMathText and LayoutDisplay return flattened text runs and rule
//     rectangles that raster or vector backends can consume.
//   - Cache and CacheConfig provide optional parsed/layout reuse. Layout
//     cache entries must be isolated by Options.MeasurementKey because
//     renderer text metrics and font resolution affect final geometry.
//
// # Typical use
//
// Implement Measurer (and optionally FontResolver, DPIMeasurer, GlyphMeasurer)
// against your rendering backend, then call LayoutMathText or LayoutDisplay
// to obtain a MathTextLayout. Iterate its Runs to draw text and Rules to fill
// rectangles. All offsets and rectangles are relative to the expression's
// baseline, with y-up.
//
// For callers that only need a plain-text Unicode rendering (no glyph
// positioning), Normalize and NormalizeDisplay return a best-effort fallback
// string without touching the Measurer interface.
//
// See the package examples for runnable snippets.
package mathtext
