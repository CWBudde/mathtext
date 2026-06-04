package mathtext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLayoutMathTextCacheReusesMeasuredLayout(t *testing.T) {
	cache := NewCache()
	measurer := &countingMeasurer{scale: 0.5}
	opts := Options{Cache: cache, MeasurementKey: "renderer-a"}

	first, ok := LayoutMathText(measurer, `\frac{1}{2}`, 20, "base", opts)
	if !ok {
		t.Fatal("first LayoutMathText returned !ok")
	}
	firstCalls := measurer.calls
	if firstCalls == 0 {
		t.Fatal("first layout did not measure text")
	}

	first.Runs[0].Text = "mutated"
	second, ok := LayoutMathText(measurer, `\frac{1}{2}`, 20, "base", opts)
	if !ok {
		t.Fatal("second LayoutMathText returned !ok")
	}
	if measurer.calls != firstCalls {
		t.Fatalf("cached layout remeasured text: first calls=%d second calls=%d", firstCalls, measurer.calls)
	}
	if second.Runs[0].Text == "mutated" {
		t.Fatalf("cached layout returned mutable run slice: %+v", second.Runs)
	}
	parsed, layouts := cache.Stats()
	if parsed != 1 || layouts != 1 {
		t.Fatalf("cache stats = parsed %d layouts %d, want 1/1", parsed, layouts)
	}
}

func TestLayoutMathTextCacheSeparatesMeasurementKeys(t *testing.T) {
	cache := NewCache()
	narrow := &countingMeasurer{scale: 0.4}
	wide := &countingMeasurer{scale: 0.8}

	narrowLayout, ok := LayoutMathText(narrow, `ab`, 20, "base", Options{Cache: cache, MeasurementKey: "narrow"})
	if !ok {
		t.Fatal("narrow LayoutMathText returned !ok")
	}
	wideLayout, ok := LayoutMathText(wide, `ab`, 20, "base", Options{Cache: cache, MeasurementKey: "wide"})
	if !ok {
		t.Fatal("wide LayoutMathText returned !ok")
	}
	if wideLayout.Width <= narrowLayout.Width {
		t.Fatalf("measurement keys reused incompatible layout: narrow=%v wide=%v", narrowLayout.Width, wideLayout.Width)
	}
	parsed, layouts := cache.Stats()
	if parsed != 1 || layouts != 2 {
		t.Fatalf("cache stats = parsed %d layouts %d, want 1/2", parsed, layouts)
	}
}

func TestLayoutMathTextCacheWithoutMeasurementKeyOnlyCachesParse(t *testing.T) {
	cache := NewCache()
	measurer := &countingMeasurer{scale: 0.5}
	opts := Options{Cache: cache}

	if _, ok := LayoutMathText(measurer, `ab`, 20, "base", opts); !ok {
		t.Fatal("first LayoutMathText returned !ok")
	}
	firstCalls := measurer.calls
	if _, ok := LayoutMathText(measurer, `ab`, 20, "base", opts); !ok {
		t.Fatal("second LayoutMathText returned !ok")
	}
	if measurer.calls <= firstCalls {
		t.Fatalf("layout cache was used without measurement key: first=%d second=%d", firstCalls, measurer.calls)
	}
	parsed, layouts := cache.Stats()
	if parsed != 1 || layouts != 0 {
		t.Fatalf("cache stats = parsed %d layouts %d, want 1/0", parsed, layouts)
	}
}

func TestLayoutMathTextCacheEvictsOldestEntriesWhenBounded(t *testing.T) {
	cache := NewCacheWithConfig(CacheConfig{MaxParsed: 1, MaxLayouts: 1})
	measurer := &countingMeasurer{scale: 0.5}

	if _, ok := LayoutMathText(measurer, `ab`, 20, "base", Options{Cache: cache, MeasurementKey: "renderer"}); !ok {
		t.Fatal("first LayoutMathText returned !ok")
	}
	if _, ok := LayoutMathText(measurer, `cd`, 20, "base", Options{Cache: cache, MeasurementKey: "renderer"}); !ok {
		t.Fatal("second LayoutMathText returned !ok")
	}
	parsed, layouts := cache.Stats()
	if parsed != 1 || layouts != 1 {
		t.Fatalf("bounded cache stats = parsed %d layouts %d, want 1/1", parsed, layouts)
	}

	calls := measurer.calls
	if _, ok := LayoutMathText(measurer, `ab`, 20, "base", Options{Cache: cache, MeasurementKey: "renderer"}); !ok {
		t.Fatal("third LayoutMathText returned !ok")
	}
	if measurer.calls <= calls {
		t.Fatalf("oldest layout entry was reused after eviction: before=%d after=%d", calls, measurer.calls)
	}
}

func TestCacheSaveLoadFileReusesLayoutAcrossProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mathtext-cache.json")
	cache := NewCache()
	measurer := &countingMeasurer{scale: 0.5}

	first, ok := LayoutMathText(measurer, `\frac{1}{2}`, 20, "base", Options{
		Cache:          cache,
		MeasurementKey: "renderer",
	})
	if !ok {
		t.Fatal("first LayoutMathText returned !ok")
	}
	if measurer.calls == 0 {
		t.Fatal("first layout did not measure text")
	}
	if err := cache.SaveFile(path); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	firstBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := cache.SaveFile(path); err != nil {
		t.Fatalf("second SaveFile: %v", err)
	}
	secondBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("second ReadFile: %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("SaveFile output is not deterministic")
	}

	loaded := NewCache()
	if err := loaded.LoadFile(path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	measurer.calls = 0
	second, ok := LayoutMathText(measurer, `\frac{1}{2}`, 20, "base", Options{
		Cache:          loaded,
		MeasurementKey: "renderer",
	})
	if !ok {
		t.Fatal("second LayoutMathText returned !ok")
	}
	if measurer.calls != 0 {
		t.Fatalf("loaded layout cache remeasured text: calls=%d", measurer.calls)
	}
	if second.Width != first.Width || len(second.Rules) != len(first.Rules) || len(second.Runs) != len(first.Runs) {
		t.Fatalf("loaded layout mismatch: first=%+v second=%+v", first, second)
	}
}
