// Package boxes draws a `fitness --json` report as a static HTML diagram with
// one box per check, as docs/ci.md#box-diagram describes.
package boxes

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// Report is the `fitness --json` record of one run: how long it took, how
// big the runner is, and each check's cost. CI keeps it as an artifact, so
// the trail of runs shows which checks slow down or grow.
type Report struct {
	ElapsedMs   int64   `json:"elapsedMs"`
	RunnerBytes int64   `json:"runnerBytes"`
	Checks      []Check `json:"checks"`
}

// Check is one check's cost: wall time against its budget, files judged,
// binary size, and error lines reported.
type Check struct {
	Name         string `json:"name"`
	Ok           bool   `json:"ok"`
	Ms           int64  `json:"ms"`
	TimeoutMs    int64  `json:"timeoutMs"`
	TimedOut     bool   `json:"timedOut"`
	FilesChecked int    `json:"filesChecked"`
	BinaryBytes  int64  `json:"binaryBytes"`
	ErrorLines   int    `json:"errorLines"`
}

// Parse decodes a report, refusing JSON that holds no checks, since a
// diagram of nothing hides a broken run.
func Parse(data []byte) (Report, error) {
	var r Report
	if err := json.Unmarshal(data, &r); err != nil {
		return Report{}, fmt.Errorf("not a fitness --json report: %w", err)
	}
	if len(r.Checks) == 0 {
		return Report{}, errors.New("not a fitness --json report: it lists no checks")
	}
	return r, nil
}

// measure is one way to size the boxes.
type measure struct {
	ID, Label, Empty string
	value            func(Check) int64
	format           func(int64) string
}

// measures are the diagram's three views, in the order the page offers them.
var measures = []measure{
	{ID: "time", Label: "Runtime", Empty: "took no measurable time", value: func(c Check) int64 { return c.Ms }, format: formatMs},
	{ID: "size", Label: "Binary size", Empty: "have no binary on disk", value: func(c Check) int64 { return c.BinaryBytes }, format: formatBytes},
	{ID: "errors", Label: "Error lines", Empty: "reported no error lines", value: func(c Check) int64 { return int64(c.ErrorLines) }, format: formatErrors},
}

// diagramWidth and diagramHeight give the layout the page's 16:10 shape, so
// boxes stay square on screen.
const diagramWidth, diagramHeight = 160.0, 100.0

// view is one measure laid out, a box per valued check plus equal small boxes
// below for checks with none.
type view struct {
	measure
	Boxes []box
	Zero  []box
}

// box is one check drawn in a view.
type box struct {
	Check
	Rect
	Value string
	// Labeled marks a box big enough to print its name inside.
	Labeled bool
}

// layoutView sizes every check by m, largest first.
func layoutView(checks []Check, m measure) view {
	sorted := slices.Clone(checks)
	slices.SortStableFunc(sorted, func(a, b Check) int { return cmp.Compare(m.value(b), m.value(a)) })
	v := view{measure: m}
	var values []float64
	for _, c := range sorted {
		if m.value(c) <= 0 {
			v.Zero = append(v.Zero, box{Check: c, Value: m.format(0)})
			continue
		}
		values = append(values, float64(m.value(c)))
		v.Boxes = append(v.Boxes, box{Check: c, Value: m.format(m.value(c))})
	}
	for i, r := range layout(values, diagramWidth, diagramHeight) {
		v.Boxes[i].Rect = percent(r)
		v.Boxes[i].Labeled = v.Boxes[i].Width >= 6 && v.Boxes[i].Height >= 6
	}
	return v
}

// percent converts a layout rect to percent of the diagram.
func percent(r Rect) Rect {
	return Rect{
		Left:   r.Left * 100 / diagramWidth,
		Top:    r.Top * 100 / diagramHeight,
		Width:  r.Width * 100 / diagramWidth,
		Height: r.Height * 100 / diagramHeight,
	}
}

func formatMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%d ms", ms)
	}
	return fmt.Sprintf("%.1f s", float64(ms)/1000)
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func formatErrors(n int64) string {
	if n == 1 {
		return "1 error line"
	}
	return fmt.Sprintf("%d error lines", n)
}

// Result names a check's outcome in words, so color never carries it alone.
func (c Check) Result() string {
	switch {
	case c.TimedOut:
		return "timed out"
	case c.Ok:
		return "passed"
	}
	return "failed"
}
