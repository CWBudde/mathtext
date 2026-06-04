// Package mathtext parses and lays out Matplotlib-style MathText expressions
// for renderer-independent drawing.
//
// The package intentionally exposes a small renderer-neutral API surface:
//   - Normalize and NormalizeDisplay provide Unicode fallback text.
//   - SplitDisplaySegments separates plain text from inline $...$ math.
//   - LayoutMathText and LayoutDisplay return flattened text runs and rule
//     rectangles that raster or vector backends can consume.
//   - Cache and CacheConfig provide optional parsed/layout reuse. Layout cache
//     entries must be isolated by MeasurementKey because renderer text metrics
//     and font resolution affect final geometry.
package mathtext
