# mathtext

Go package for parsing and laying out Matplotlib-style MathText expressions.

The package is **renderer-neutral**: callers provide text measurement through
the `Measurer` interface and optional font selection through `FontResolver`,
then draw the returned text runs and rule rectangles with whatever rendering
backend they like (raster, SVG, PDF, OpenGL, ...).

## Install

```sh
go get github.com/cwbudde/mathtext
```

Requires Go 1.22 or newer.

## Quick start

```go
package main

import (
    "fmt"

    "github.com/cwbudde/mathtext"
)

type stubMeasurer struct{}

func (stubMeasurer) MeasureText(text string, size float64, _ string) mathtext.Metrics {
    return mathtext.Metrics{
        W:       float64(len([]rune(text))) * size * 0.5,
        H:       size,
        Ascent:  size * 0.8,
        Descent: size * 0.2,
    }
}

func main() {
    layout, ok := mathtext.LayoutMathText(stubMeasurer{}, `\alpha_i^2`, 12, "", mathtext.Options{})
    if !ok {
        return
    }
    fmt.Printf("width=%.1f ascent=%.1f descent=%.1f\n", layout.Width, layout.Ascent, layout.Descent)
    for _, run := range layout.Runs {
        fmt.Printf("  run %q at (%.1f, %.1f) size=%.1f\n", run.Text, run.Offset.X, run.Offset.Y, run.FontSize)
    }
}
```

For a non-layout, plain-text Unicode fallback use `Normalize` or
`NormalizeDisplay`:

```go
mathtext.NormalizeDisplay(`signal $\alpha_i^2$ peak`)
// → "signal αᵢ² peak"
```

See the [`examples/`](./examples) directory for runnable CLIs and
`example_test.go` for documented godoc examples.

## What's supported

The parser handles the practical subset of TeX/Matplotlib MathText that
shows up in plot labels and titles:

- Greek letters (`\alpha`, `\Omega`, ...) and many TeX symbol commands
- Sub- and superscripts (`a_i^2`)
- Fractions: `\frac`, `\dfrac`, `\genfrac`
- Square roots: `\sqrt`, `\sqrt[n]`
- Stretchy delimiters: `\left( ... \right)`, `\bigl`, `\Bigr`, etc.
- Spacing commands: `\,`, `\:`, `\;`, `\!`, `\quad`, `\qquad`
- Math styles: `\mathit`, `\mathrm`, `\mathbf`, `\mathsf`, `\mathtt`,
  `\mathcal`, `\mathbb`
- Matrix environments: `pmatrix`, `bmatrix`, `vmatrix`, `Vmatrix`, `matrix`
- Accents (centred separate-glyph over the nucleus): `\hat`, `\bar`, `\vec`,
  `\dot`, `\ddot`, `\dddot`, `\ddddot`, `\tilde`, `\breve`, `\grave`, `\acute`,
  `\mathring`, `\overrightarrow`, `\overleftarrow`, and the char forms
  `\^ \~ \' \. \" \``
- Wide accents (scaled to the nucleus width): `\widehat`, `\widetilde`,
  `\widebar`
- Overline rule: `\overline`
- Stacking: `\overset`, `\underset`, `\stackrel`, and `\substack{a \\ b}`
- Inline math segments inside display text via `$...$`

Beyond Matplotlib's MathText, the parser also accepts `\overbrace`,
`\underbrace`, and `\not` as best-effort extensions (no Matplotlib reference).

The full command tables live in
[`normalize.go`](./normalize.go) — `mathTextCommandMap`,
`mathTextOperatorMap`, `mathTextSpacingCommands`, and friends.

## Layout caching

`Options.Cache` is opt-in. The parsed-expression cache is safe to share
across callers, but **layout** cache entries depend on the renderer's text
metrics and font resolution, so the cache only stores them when you pass a
non-empty `Options.MeasurementKey` that uniquely identifies the active
`Measurer` + `FontResolver` configuration. Different DPIs, font sets, or
hinting choices must use different keys.

```go
cache := mathtext.NewCache()
opts := mathtext.Options{
    Cache:          cache,
    MeasurementKey: "freetype@96dpi",
}
layout, _ := mathtext.LayoutMathText(measurer, `\frac{a}{b}`, 12, "", opts)
```

`Cache.SaveFile` / `Cache.LoadFile` persist the parsed-node cache to disk so
repeated invocations of short-lived tools can skip re-parsing.

## Origin

This package was split from the internal `mathtext` package of
[matplotlib-go](https://github.com/MeKo-Tech/matplotlib-go) so it can be
versioned and consumed independently of any one rendering backend.
