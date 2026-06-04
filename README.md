# mathtext

Standalone Go package for parsing and laying out Matplotlib-style MathText.

The package is renderer-neutral: callers provide text measurement through
`Measurer` and optional font resolution through `FontResolver`, then draw the
returned text runs and rule rectangles with their own renderer.

This repository was split from a plotting library internal package for independent versioning.
