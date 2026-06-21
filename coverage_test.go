package mathtext

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
	"unicode"
)

// symbolCoverageTarget is the fraction of matplotlib's tex2uni command names the
// library must resolve to a glyph (rather than echoing the literal command).
const symbolCoverageTarget = 0.95

// mathSymbolResolvable reports whether the named TeX command (without the
// leading backslash) maps to a glyph through any of the library's tables or the
// parser's single-character escape handling. This mirrors the lookups in
// parseCommand / parseCommandNode.
func mathSymbolResolvable(name string) bool {
	if name == "" {
		return false
	}
	if _, ok := tex2uni[name]; ok {
		return true
	}
	if _, ok := mathTextCommandMap[name]; ok {
		return true
	}
	if _, ok := mathTextDelimiterCommands[name]; ok {
		return true
	}
	if _, ok := mathTextOperatorMap[name]; ok {
		return true
	}
	if _, ok := mathTextSpacingCommands[name]; ok {
		return true
	}
	if _, ok := mathTextSpacingCommandWidths[name]; ok {
		return true
	}
	if _, ok := mathTextAccentMarks[name]; ok {
		return true
	}
	if _, ok := mathTextPassthroughCommands[name]; ok {
		return true
	}
	if mathAlphabetCommandName(name) {
		return true
	}
	// Single non-letter escapes (\#, \{, \|, …) are emitted verbatim by the
	// parser's non-letter branch.
	if r := []rune(name); len(r) == 1 && !unicode.IsLetter(r[0]) {
		return true
	}
	return false
}

// TestSymbolCoverage measures and reports how much of matplotlib's tex2uni
// symbol table the library resolves. The expected name list is vendored from
// matplotlib 3.10.9 in testdata/tex2uni_symbols.json (see testdata/README.md).
func TestSymbolCoverage(t *testing.T) {
	raw, err := os.ReadFile("testdata/tex2uni_symbols.json")
	if err != nil {
		t.Fatalf("read symbol table: %v", err)
	}
	var data struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("decode symbol table: %v", err)
	}
	if len(data.Names) == 0 {
		t.Fatal("symbol table is empty")
	}

	var missing []string
	for _, name := range data.Names {
		if !mathSymbolResolvable(name) {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)

	resolved := len(data.Names) - len(missing)
	coverage := float64(resolved) / float64(len(data.Names))
	t.Logf("symbol coverage: %d/%d = %.1f%%", resolved, len(data.Names), coverage*100)
	if len(missing) > 0 {
		shown := missing
		if len(shown) > 30 {
			shown = shown[:30]
		}
		t.Logf("unresolved (%d): %v", len(missing), shown)
	}

	if coverage < symbolCoverageTarget {
		t.Errorf("symbol coverage %.1f%% below target %.1f%%", coverage*100, symbolCoverageTarget*100)
	}
}
