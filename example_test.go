package mathtext_test

import (
	"fmt"

	"github.com/cwbudde/mathtext"
)

type docMeasurer struct{}

func (docMeasurer) MeasureText(text string, size float64, _ string) mathtext.Metrics {
	return mathtext.Metrics{
		W:       float64(len([]rune(text))) * size * 0.5,
		H:       size,
		Ascent:  size * 0.8,
		Descent: size * 0.2,
	}
}

func ExampleNormalize() {
	fmt.Println(mathtext.Normalize(`\alpha_i^2`))
	fmt.Println(mathtext.Normalize(`\frac{1}{2}`))
	// Output:
	// αᵢ²
	// 1⁄2
}

func ExampleNormalizeDisplay() {
	fmt.Println(mathtext.NormalizeDisplay(`signal $\alpha_i^2$ peak`))
	// Output: signal αᵢ² peak
}

func ExampleSplitDisplaySegments() {
	segments, hasMath, ok := mathtext.SplitDisplaySegments(`E = $mc^2$ today`)
	fmt.Println(ok, hasMath)
	for _, s := range segments {
		fmt.Printf("math=%-5v %q\n", s.IsMath, s.Text)
	}
	// Output:
	// true true
	// math=false "E = "
	// math=true  "mc^2"
	// math=false " today"
}

func ExampleLayoutMathText() {
	layout, ok := mathtext.LayoutMathText(docMeasurer{}, `a+b`, 10, "", mathtext.Options{})
	if !ok {
		return
	}
	fmt.Printf("width=%.2f ascent=%.2f descent=%.2f\n", layout.Width, layout.Ascent, layout.Descent)
	for _, run := range layout.Runs {
		fmt.Printf("run %q @ x=%.2f size=%.2f\n", run.Text, run.Offset.X, run.FontSize)
	}
	// Output:
	// width=17.00 ascent=8.00 descent=2.00
	// run "a" @ x=0.00 size=10.00
	// run "+" @ x=6.00 size=10.00
	// run "b" @ x=12.00 size=10.00
}
