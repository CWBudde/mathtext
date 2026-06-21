package mathtext

import "testing"

func TestUnknownCommandReportedToHandler(t *testing.T) {
	var got []string
	restore := SetUnknownCommandHandler(func(name string) { got = append(got, name) })
	defer restore()

	// nil cache: always parse fresh so the handler is exercised.
	_ = parseMathLayoutNode(`\zzzdefinitelyunknown x`, nil)

	if len(got) != 1 || got[0] != "zzzdefinitelyunknown" {
		t.Fatalf("unknown command handler got %v, want [zzzdefinitelyunknown]", got)
	}
}

func TestKnownCommandNotReported(t *testing.T) {
	var got []string
	restore := SetUnknownCommandHandler(func(name string) { got = append(got, name) })
	defer restore()

	_ = parseMathLayoutNode(`\frac{1}{2}`, nil)

	if len(got) != 0 {
		t.Fatalf("known commands must not be reported as unknown, got %v", got)
	}
}

func TestSetUnknownCommandHandlerRestores(t *testing.T) {
	restore := SetUnknownCommandHandler(func(string) { t.Fatal("outer handler should have been replaced") })
	inner := 0
	restoreInner := SetUnknownCommandHandler(func(string) { inner++ })
	_ = parseMathLayoutNode(`\nope`, nil)
	restoreInner()
	restore()
	if inner != 1 {
		t.Fatalf("inner handler call count = %d, want 1", inner)
	}
}
