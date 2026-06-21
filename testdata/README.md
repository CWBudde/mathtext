# testdata

## tex2uni_symbols.json

The list of TeX command names from matplotlib's `tex2uni` table (632 entries),
used as the parity target by `TestSymbolCoverage`. It is the source of truth for
`tex_tables.go` as well.

Regenerated from the matplotlib snapshot vendored in the matplotlib-go repo
(`third_party/matplotlib`, version 3.10.9). The extraction script loads
`lib/matplotlib/_mathtext_data.py` in isolation (importing the full matplotlib
package shadows the stdlib), emits `tex_tables.go` and this JSON, then the Go
files are formatted with `gofumpt`:

```python
import importlib.util, json
base = '<matplotlib-go>/third_party/matplotlib/lib/matplotlib'
spec = importlib.util.spec_from_file_location('mtd', base + '/_mathtext_data.py')
m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
json.dump({'names': sorted(m.tex2uni)}, open('tex2uni_symbols.json', 'w'),
          indent=1, ensure_ascii=True)
```

The spacing-class sets in `tex_tables.go` (`mathTex2UniSpacedNames`) are the
intersection of `tex2uni` with matplotlib's `_binary_operators`,
`_relation_symbols`, and `_arrow_symbols` from `lib/matplotlib/_mathtext.py`.
