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
