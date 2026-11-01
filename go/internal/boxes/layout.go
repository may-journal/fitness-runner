package boxes

// Rect is one box's place inside the diagram, in percent of its width and
// height, so the page scales it without recomputing anything.
type Rect struct {
	Left, Top, Width, Height float64
}

// layout splits an area into one rect per value, positive and sorted largest
// first, each sized in proportion and kept near square.
func layout(values []float64, width, height float64) []Rect {
	total := sum(values)
	if total <= 0 {
		return nil
	}
	scale := width * height / total
	areas := make([]float64, len(values))
	for i, v := range values {
		areas[i] = v * scale
	}
	rects := make([]Rect, 0, len(values))
	free := Rect{Width: width, Height: height}
	for len(areas) > 0 {
		n := rowLength(areas, min(free.Width, free.Height))
		var placed []Rect
		placed, free = placeRow(areas[:n], free)
		rects = append(rects, placed...)
		areas = areas[n:]
	}
	return rects
}

// rowLength is how many leading areas share one row along a side before adding
// another makes it less square.
func rowLength(areas []float64, side float64) int {
	n := 1
	for n < len(areas) && worst(areas[:n+1], side) <= worst(areas[:n], side) {
		n++
	}
	return n
}

// worst is the largest aspect ratio among a row's boxes laid along side.
func worst(row []float64, side float64) float64 {
	s := sum(row)
	big, small := row[0], row[len(row)-1]
	return max(side*side*big/(s*s), s*s/(side*side*small))
}

// placeRow lays a row of areas along the shorter side of free and returns
// their rects and the space left over.
func placeRow(row []float64, free Rect) ([]Rect, Rect) {
	s := sum(row)
	rects := make([]Rect, len(row))
	if free.Width >= free.Height {
		thick := s / free.Height
		top := free.Top
		for i, a := range row {
			rects[i] = Rect{Left: free.Left, Top: top, Width: thick, Height: a / thick}
			top += a / thick
		}
		return rects, Rect{Left: free.Left + thick, Top: free.Top, Width: free.Width - thick, Height: free.Height}
	}
	thick := s / free.Width
	left := free.Left
	for i, a := range row {
		rects[i] = Rect{Left: left, Top: free.Top, Width: a / thick, Height: thick}
		left += a / thick
	}
	return rects, Rect{Left: free.Left, Top: free.Top + thick, Width: free.Width, Height: free.Height - thick}
}

func sum(values []float64) float64 {
	var s float64
	for _, v := range values {
		s += v
	}
	return s
}
