// mathlayout is a small CLI that lays out one MathText expression using a
// stub Measurer and prints the resulting runs and rules. It demonstrates the
// renderer-neutral integration without requiring a real font backend.
//
// Usage:
//
//	go run ./examples/mathlayout '\frac{a}{b}'
package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

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
	size := flag.Float64("size", 12, "font size in points")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-size N] EXPRESSION\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	expr := flag.Arg(0)

	layout, ok := mathtext.LayoutMathText(stubMeasurer{}, expr, *size, "", mathtext.Options{})
	if !ok {
		fmt.Fprintf(os.Stderr, "layout failed for expression: %q\n", expr)
		os.Exit(1)
	}

	fmt.Printf("Expression: %s\n", expr)
	fmt.Printf("Width:   %.3f\n", layout.Width)
	fmt.Printf("Ascent:  %.3f\n", layout.Ascent)
	fmt.Printf("Descent: %.3f\n", layout.Descent)
	fmt.Printf("Height:  %.3f\n", layout.Height)

	if len(layout.Runs) > 0 {
		fmt.Println("\nRuns:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  #\tTEXT\tX\tY\tSIZE\tFONT")
		for i, r := range layout.Runs {
			fmt.Fprintf(w, "  %d\t%q\t%.3f\t%.3f\t%.3f\t%s\n",
				i, r.Text, r.Offset.X, r.Offset.Y, r.FontSize, r.FontKey)
		}
		_ = w.Flush()
	}

	if len(layout.Rules) > 0 {
		fmt.Println("\nRules:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  #\tX0\tY0\tX1\tY1\tW\tH")
		for i, r := range layout.Rules {
			fmt.Fprintf(w, "  %d\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\n",
				i, r.Rect.Min.X, r.Rect.Min.Y, r.Rect.Max.X, r.Rect.Max.Y, r.Rect.W(), r.Rect.H())
		}
		_ = w.Flush()
	}
}
