package mathtext

import "testing"

func TestNormalizeDisplayParsesInlineMath(t *testing.T) {
	got := NormalizeDisplay(`signal $\\alpha_i^2$ peak`)
	if got != "signal αᵢ² peak" {
		t.Fatalf("NormalizeDisplay = %q, want %q", got, "signal αᵢ² peak")
	}
}

func TestSplitDisplaySegmentsRejectsUnbalancedMath(t *testing.T) {
	if _, _, ok := SplitDisplaySegments(`cost is $5`); ok {
		t.Fatal("SplitDisplaySegments returned ok for unbalanced math")
	}
}

func TestNormalizeDisplayHandlesExplicitSpacingCommands(t *testing.T) {
	got := NormalizeDisplay(`$a\\hspace{0.5em}b\\negthinspace c$`)
	if got != "a b c" {
		t.Fatalf("NormalizeDisplay = %q, want %q", got, "a b c")
	}
}

func TestNormalizeDisplayMatrixEnvironment(t *testing.T) {
	// Fully-doubled source: the row separator arrives as `\\\\` and must collapse
	// to a single row break, not two (no spurious empty row).
	got := NormalizeDisplay(`$\\begin{pmatrix} a & b \\\\ c & d \\end{pmatrix}$`)
	if got != "(a b; c d)" {
		t.Fatalf("NormalizeDisplay = %q, want %q", got, "(a b; c d)")
	}
}

func TestNormalizeDisplayMatrixSingleSeparator(t *testing.T) {
	// A raw (non-doubled) `\\` separator must still produce one row break.
	got := NormalizeDisplay(`$\begin{pmatrix} a & b \\ c & d \end{pmatrix}$`)
	if got != "(a b; c d)" {
		t.Fatalf("NormalizeDisplay = %q, want %q", got, "(a b; c d)")
	}
}
