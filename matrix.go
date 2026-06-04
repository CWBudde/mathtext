package mathtext

func layoutMathMatrix(r Measurer, n mathLayoutNode, size float64, fontKey string, opts Options) mathLayoutBox {
	if len(n.rows) == 0 {
		return mathLayoutBox{}
	}

	cellBoxes := make([][]mathLayoutBox, len(n.rows))
	numCols := 0
	for i, row := range n.rows {
		cellBoxes[i] = make([]mathLayoutBox, len(row))
		if len(row) > numCols {
			numCols = len(row)
		}
		for j, cell := range row {
			cellBoxes[i][j] = layoutMathNode(r, cell, size, fontKey, opts)
		}
	}
	if numCols == 0 {
		return mathLayoutBox{}
	}

	colWidths := make([]float64, numCols)
	rowAscents := make([]float64, len(n.rows))
	rowDescents := make([]float64, len(n.rows))
	for i, row := range cellBoxes {
		for j, cell := range row {
			if cell.Width > colWidths[j] {
				colWidths[j] = cell.Width
			}
			if cell.Ascent > rowAscents[i] {
				rowAscents[i] = cell.Ascent
			}
			if cell.Descent > rowDescents[i] {
				rowDescents[i] = cell.Descent
			}
		}
		if rowAscents[i] == 0 && rowDescents[i] == 0 {
			rowAscents[i] = size * 0.5
			rowDescents[i] = size * 0.3
		}
	}

	colGap := size * 0.6
	rowGap := size * 0.4
	bodyWidth := 0.0
	for i, width := range colWidths {
		bodyWidth += width
		if i > 0 {
			bodyWidth += colGap
		}
	}
	bodyHeight := 0.0
	for i := range n.rows {
		bodyHeight += rowAscents[i] + rowDescents[i]
		if i > 0 {
			bodyHeight += rowGap
		}
	}

	left := layoutMathDelimiter(r, n.left, bodyHeight/2, bodyHeight/2, size, fontKey)
	right := layoutMathDelimiter(r, n.right, bodyHeight/2, bodyHeight/2, size, fontKey)
	leftGap := 0.0
	rightGap := 0.0
	if left.Width > 0 {
		leftGap = size * 0.18
	}
	if right.Width > 0 {
		rightGap = size * 0.18
	}

	var out mathLayoutBox
	x := 0.0
	out.appendTranslated(left, x, 0)
	x += left.Width
	if left.Width > 0 {
		x += leftGap
	}

	top := -bodyHeight / 2
	for i, row := range cellBoxes {
		baselineY := top + rowAscents[i]
		cellX := x
		for j := 0; j < numCols; j++ {
			var cell mathLayoutBox
			if j < len(row) {
				cell = row[j]
			}
			cellOffsetX := cellX + (colWidths[j]-cell.Width)/2
			out.appendTranslated(cell, cellOffsetX, baselineY)
			cellX += colWidths[j] + colGap
		}
		top += rowAscents[i] + rowDescents[i] + rowGap
	}
	x += bodyWidth
	if right.Width > 0 {
		x += rightGap
	}
	out.appendTranslated(right, x, 0)
	out.Width = left.Width + leftGap + bodyWidth + rightGap + right.Width
	out.Ascent = bodyHeight / 2
	out.Descent = bodyHeight / 2
	if left.Ascent > out.Ascent {
		out.Ascent = left.Ascent
	}
	if right.Ascent > out.Ascent {
		out.Ascent = right.Ascent
	}
	if left.Descent > out.Descent {
		out.Descent = left.Descent
	}
	if right.Descent > out.Descent {
		out.Descent = right.Descent
	}
	return out
}

func matrixEnvironmentDelimiters(name string) (left, right string, ok bool) {
	switch name {
	case "matrix", "array":
		return "", "", true
	case "pmatrix":
		return "(", ")", true
	case "bmatrix":
		return "[", "]", true
	case "Bmatrix":
		return "{", "}", true
	case "vmatrix":
		return "|", "|", true
	case "Vmatrix":
		return "‖", "‖", true
	default:
		return "", "", false
	}
}
