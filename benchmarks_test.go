package mathtext

import "testing"

const (
	benchInlineMath  = `\alpha_i^2 + \beta_{j}^{3} - \gamma_{kl} \cdot \delta`
	benchDisplayText = `signal $\alpha_i^2$ peak at $\frac{\omega_0}{2\pi}$ Hz`
	benchFraction    = `\frac{a+b}{c-d} + \sqrt{x^2 + y^2}`
)

func BenchmarkNormalize(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Normalize(benchInlineMath)
	}
}

func BenchmarkNormalizeDisplay(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NormalizeDisplay(benchDisplayText)
	}
}

func BenchmarkSplitDisplaySegments(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = SplitDisplaySegments(benchDisplayText)
	}
}

func BenchmarkLayoutMathTextCold(b *testing.B) {
	m := testMeasurer{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := LayoutMathText(m, benchFraction, 12, "", Options{}); !ok {
			b.Fatal("layout failed")
		}
	}
}

func BenchmarkLayoutMathTextCached(b *testing.B) {
	m := testMeasurer{}
	opts := Options{Cache: NewCache(), MeasurementKey: "bench"}
	if _, ok := LayoutMathText(m, benchFraction, 12, "", opts); !ok {
		b.Fatal("warmup layout failed")
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := LayoutMathText(m, benchFraction, 12, "", opts); !ok {
			b.Fatal("layout failed")
		}
	}
}

func BenchmarkLayoutDisplay(b *testing.B) {
	m := testMeasurer{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := LayoutDisplay(m, benchDisplayText, 12, "", Options{}); !ok {
			b.Fatal("layout failed")
		}
	}
}
