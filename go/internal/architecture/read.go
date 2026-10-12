package architecture

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mermaid"
	"github.com/may-journal/fitness-runner/go/internal/requirement"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

var views = map[string]string{
	"01-system-context.md": "context", "02-containers.md": "containers",
	"03-components.md": "components", "04-code.md": "code", "05-user-journey.md": "journey",
}

var heading = regexp.MustCompile(`^#{1,6}\s+(.*?)\s*#*$`)

// Load reads every view and its connected sources, including in scoped runs,
// so a change on one side still meets the unchanged other side.
func Load(root string) (Model, error) {
	files := walkfs.FilesByExt(root, "")
	m := Model{repo: newRepository(root, files)}
	if err := m.readRequirements(root, files); err != nil {
		return m, err
	}
	present, err := m.readViews(root, files)
	if err != nil || len(m.Files) == 0 {
		return Model{}, err
	}
	m.missingViews(present)
	m.actors()
	m.inventory(root, files)
	return m, nil
}

func (m *Model) readViews(root string, files []string) (map[string]bool, error) {
	present := map[string]bool{}
	for _, file := range files {
		view := viewFile(file)
		if view == "" {
			continue
		}
		present[view] = true
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return present, err
		}
		m.Files = append(m.Files, file)
		m.readView(file, view, string(body))
	}
	return present, nil
}

func viewFile(file string) string {
	dir := path.Dir(file)
	if dir == "architecture" || dir == "docs/architecture" {
		return views[path.Base(file)]
	}
	return ""
}

func (m *Model) missingViews(present map[string]bool) {
	for _, file := range sortedKeys(views) {
		if !present[views[file]] {
			m.fail(Location{m.Files[0], 1}, "missing architecture view %s", file)
		}
	}
}

func (m *Model) readRequirements(root string, files []string) error {
	var paths []string
	for _, file := range files {
		if strings.HasPrefix(file, "docs/requirements/") && strings.HasSuffix(file, ".md") && path.Base(file) != "template.md" {
			paths = append(paths, file)
		}
	}
	// The requirements check owns doc shape errors; this check only resolves
	// references against the parsed docs.
	docs, _, err := requirement.Load(root, paths)
	m.Requirements = docs
	return err
}

// viewReader pairs each model diagram with the callout table after it.
// Unnumbered legends are invisible, as in the mermaid checks; a numbered
// legend keeps its own table out of the model.
type viewReader struct {
	m             *Model
	file, view    string
	lines         []string
	pending       *diagram
	skipTable     bool
	diagramsCount int
}

func (m *Model) readView(file, view, body string) {
	r := &viewReader{m: m, file: file, view: view, lines: strings.Split(body, "\n")}
	for _, block := range mermaid.ParseDoc(body) {
		r.block(block)
	}
	r.flush()
	if r.diagramsCount == 0 {
		m.fail(Location{file, 1}, "view has no diagram outside its legends")
	}
}

func (r *viewReader) block(block mermaid.DocBlock) {
	switch b := block.(type) {
	case *mermaid.DiagramBlock:
		r.diagram(b)
	case *mermaid.CalloutTableBlock:
		r.table(b)
	}
}

func (r *viewReader) diagram(b *mermaid.DiagramBlock) {
	if len(b.Numbers) == 0 {
		return
	}
	r.flush()
	if r.legend(b) {
		r.skipTable = true
		return
	}
	r.pending = r.m.readDiagram(r.file, r.view, b)
	r.diagramsCount++
}

func (r *viewReader) table(b *mermaid.CalloutTableBlock) {
	if r.skipTable {
		r.skipTable = false
		return
	}
	if r.pending != nil {
		r.pending.bindRows(b)
		r.flush()
	}
}

func (r *viewReader) flush() {
	if r.pending != nil {
		r.m.addDiagram(r.pending)
	}
	r.pending, r.skipTable = nil, false
}

// legend reports a diagram titled Legend or placed under a Legend heading.
func (r *viewReader) legend(b *mermaid.DiagramBlock) bool {
	for _, line := range strings.Split(b.Body, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "title Legend") {
			return true
		}
	}
	return strings.EqualFold(r.headingBefore(b.Line), "Legend")
}

func (r *viewReader) headingBefore(line int) string {
	for i := line - 2; i >= 0; i-- {
		if match := heading.FindStringSubmatch(r.lines[i]); match != nil {
			return match[1]
		}
	}
	return ""
}
