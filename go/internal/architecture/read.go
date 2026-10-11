package architecture

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mermaid"
	"github.com/may-journal/fitness-runner/go/internal/requirement"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

var views = map[string]string{
	"01-system-context.md": "context", "02-containers.md": "containers",
	"03-components.md": "components", "04-code.md": "code", "05-user-journey.md": "journey",
}

// Load reads all connected files, including in scoped runs. Architecture roots
// contain the five conventional view files; ADRs are not architecture views.
func Load(root string) (Model, error) {
	m := Model{}
	files := walkfs.FilesByExt(root, "")
	present, err := m.readViews(root, files)
	if err != nil {
		return m, err
	}
	if len(m.Files) == 0 {
		return m, nil
	}
	m.missingViews(present)
	if err := m.readRequirements(root, files); err != nil {
		return m, err
	}
	m.validate(root, files)
	return m, nil
}

func (m *Model) readViews(root string, files []string) (map[string]bool, error) {
	present := map[string]bool{}
	for _, file := range files {
		if view := viewFile(file); view != "" {
			present[view] = true
			if err := m.readView(root, file, view); err != nil {
				return present, err
			}
		}
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
	docs, errs, err := requirement.Load(root, paths)
	m.Requirements = docs
	m.Errors = append(m.Errors, errs...)
	return err
}

func (m *Model) readView(root, file, view string) error {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return err
	}
	m.Files = append(m.Files, file)
	m.parseView(file, view, string(b))
	return nil
}

func (m *Model) parseView(file, view, body string) {
	blocks := mermaid.ParseDoc(body)
	var pending *mermaid.DiagramBlock
	count := 0
	for _, block := range blocks {
		switch b := block.(type) {
		case *mermaid.DiagramBlock:
			pending = m.nextDiagram(file, pending, b)
		case *mermaid.CalloutTableBlock:
			count += m.pairedTable(file, view, pending, b)
			pending = nil
		}
	}
	m.unpaired(file, pending)
	if count == 0 {
		m.fail(Location{file, 1}, "view has no contracted diagram")
	}
}

func (m *Model) pairedTable(file, view string, diagram *mermaid.DiagramBlock, table *mermaid.CalloutTableBlock) int {
	if diagram == nil {
		return 0
	}
	m.readTable(file, view, diagram, table)
	return 1
}

func (m *Model) nextDiagram(file string, pending, next *mermaid.DiagramBlock) *mermaid.DiagramBlock {
	if strings.Contains(next.Body, "%% architecture:legend") {
		return pending
	}
	m.unpaired(file, pending)
	return next
}

func (m *Model) unpaired(file string, diagram *mermaid.DiagramBlock) {
	if diagram != nil {
		m.fail(Location{file, diagram.Line}, "diagram needs a Contract column; mark legends with %% architecture:legend")
	}
}

func contractColumn(header []string) int {
	for i, cell := range header {
		if cell == "Contract" {
			return i
		}
	}
	return -1
}

func (m *Model) readTable(file, view string, diagram *mermaid.DiagramBlock, table *mermaid.CalloutTableBlock) {
	column := contractColumn(table.Header)
	if column < 0 {
		m.fail(Location{file, table.Line}, "callout table needs a Contract column for diagram %s:%d", file, diagram.Line)
		return
	}
	graph := m.readDiagram(file, diagram)
	for i, row := range table.Rows {
		at := Location{file, table.Line + 2 + i}
		if column >= len(row) {
			m.fail(at, "missing Contract cell")
			continue
		}
		d := m.declaration(row[column], at)
		d.View, d.Diagram = view, Location{file, diagram.Line}
		m.matchCallout(row[0], d, graph)
		m.Declarations = append(m.Declarations, d)
	}
	m.unclaimed(graph)
}
