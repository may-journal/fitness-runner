package changelog

import "strings"

// Regions of a file a patch walks through, from its first line. The zero
// region is the body, so a step the table lacks lands there.
const (
	bodyRegion = iota
	openingRegion
	insideRegion
	tailRegion
)

// Kinds of line a patch shows on the file's old side.
const (
	otherLine = iota
	markerLine
	blankDrop
)

// regionAfter is the region a line of each kind moves the walk to. The
// opening --- leads inside the block, the closing one to the blank lines
// after it, and anything else past those reaches the body.
var regionAfter = map[[2]int]int{
	{openingRegion, markerLine}: insideRegion,
	{insideRegion, otherLine}:   insideRegion,
	{insideRegion, blankDrop}:   insideRegion,
	{insideRegion, markerLine}:  tailRegion,
	{tailRegion, blankDrop}:     tailRegion,
}

// DropsOnlyFrontMatter reports whether patch, a file's unified diff, adds
// nothing and deletes only leading front matter and the blank lines after it.
func DropsOnlyFrontMatter(patch string) bool {
	_, hunk, found := strings.Cut("\n"+patch, "\n@@ -1,")
	_, body, _ := strings.Cut(hunk, "\n")
	return found && !strings.Contains(body, "\n@@ ") && frontOnly(strings.Split(body, "\n"))
}

// frontOnly walks a hunk's lines from the file's first line and reports
// whether each one is allowed where it falls.
func frontOnly(lines []string) bool {
	region := openingRegion
	for _, line := range lines {
		var ok bool
		if region, ok = step(region, line); !ok {
			return false
		}
	}
	return true
}

// step reads one hunk line in region and returns the region it leads to.
// It reports false for an added line, or a deleted line in the body.
func step(region int, line string) (int, bool) {
	if line == "" || line[0] == '\\' {
		return region, true
	}
	next := regionAfter[[2]int{region, kindOf(line[1:], line[0] == '-')}]
	return next, line[0] != '+' && (line[0] != '-' || next != bodyRegion)
}

// kindOf sorts a line by its old-side text and whether the patch deletes it.
func kindOf(text string, deleted bool) int {
	switch {
	case strings.TrimRight(text, "\r") == "---":
		return markerLine
	case deleted && strings.TrimRight(text, "\r") == "":
		return blankDrop
	}
	return otherLine
}
