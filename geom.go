package mathtext

// Pt represents a 2D point in layout coordinates.
type Pt struct {
	X, Y float64
}

// Rect is an axis-aligned rectangle used for MathText rules.
type Rect struct {
	Min, Max Pt
}

// W returns the rectangle width.
func (r Rect) W() float64 { return r.Max.X - r.Min.X }

// H returns the rectangle height.
func (r Rect) H() float64 { return r.Max.Y - r.Min.Y }
