package mathtext

import "strings"

type testMeasurer struct{}

func (testMeasurer) MeasureText(text string, size float64, _ string) Metrics {
	return Metrics{
		W:       float64(len([]rune(text))) * size * 0.5,
		H:       size,
		Ascent:  size * 0.8,
		Descent: size * 0.2,
	}
}

type shapingMeasurer struct{}

func (shapingMeasurer) MeasureText(text string, size float64, _ string) Metrics {
	ascent := size * 0.8
	descent := size * 0.2
	return Metrics{
		W:       float64(len([]rune(text))) * size * 0.55,
		H:       ascent + descent,
		Ascent:  ascent,
		Descent: descent,
		BoundsY: -ascent,
		BoundsH: ascent + descent,
	}
}

type countingMeasurer struct {
	calls int
	scale float64
}

func (m *countingMeasurer) MeasureText(text string, size float64, _ string) Metrics {
	m.calls++
	return Metrics{
		W:       float64(len([]rune(text))) * size * m.scale,
		H:       size,
		Ascent:  size * 0.8,
		Descent: size * 0.2,
	}
}

type kerningGlyphMeasurer struct{}

func (kerningGlyphMeasurer) MeasureText(text string, size float64, _ string) Metrics {
	return Metrics{
		W:       float64(len([]rune(text))) * size,
		H:       size,
		Ascent:  size * 0.8,
		Descent: size * 0.2,
	}
}

func (kerningGlyphMeasurer) GlyphRun(text string, _ float64, _ string) ([]GlyphInfo, bool) {
	out := make([]GlyphInfo, 0, len([]rune(text)))
	for _, r := range text {
		info := GlyphInfo{Advance: 10, Iceberg: 8, Height: 10, Xmax: 10, Ymin: -2, Ymax: 8}
		if r == 'e' {
			info.KernToPrev = -3
		}
		if r == ' ' {
			info.Advance = 5
			info.Xmax = 5
		}
		out = append(out, info)
	}
	return out, true
}

type recordingResolver struct {
	requests []FontRequest
}

func (r *recordingResolver) ResolveMathFontKey(_ string, request FontRequest) string {
	r.requests = append(r.requests, request)
	if len(request.Families) > 0 {
		return "resolved:" + strings.Join(request.Families, ",")
	}
	if request.Style != "" {
		return "style:" + string(request.Style)
	}
	return ""
}

type shapingFontResolver struct{}

func (shapingFontResolver) ResolveMathFontKey(base string, request FontRequest) string {
	if len(request.Families) > 0 {
		return strings.Join(request.Families, ", ")
	}
	if request.Style != "" {
		return string(request.Style) + ":" + base
	}
	if request.Weight > 0 {
		return "weight:" + base
	}
	return base
}

func isTestSTIXSizeFont(fontKey string) bool {
	return strings.HasPrefix(fontKey, "STIXSize")
}

func containsTestRun(runs []MathTextLayoutRun, text string, size float64) bool {
	for _, run := range runs {
		if run.Text == text && run.FontSize == size {
			return true
		}
	}
	return false
}

func containsRunWithFont(runs []MathTextLayoutRun, text, fontKey string) bool {
	for _, run := range runs {
		if run.Text == text && run.FontKey == fontKey {
			return true
		}
	}
	return false
}
