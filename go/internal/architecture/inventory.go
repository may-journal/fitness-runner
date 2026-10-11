package architecture

import (
	"go/parser"
	"go/token"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var sourceExtensions = strings.Fields(".go .swift .js .jsx .ts .tsx .mjs .cjs .py .sh .bash .zsh .c .h .cc .cpp .rs .java .kt .m .mm")

func positiveNumber(text string) int {
	n, err := strconv.Atoi(text)
	if err != nil || n < 1 {
		return 0
	}
	return n
}

func sourceCategory(file string) string {
	lower := strings.ToLower(file)
	if strings.Contains(lower, "/test") || strings.HasSuffix(lower, "_test.go") || strings.HasPrefix(lower, "tests/") {
		return "test"
	}
	if contains([]string{".sh", ".bash", ".zsh", ".py"}, path.Ext(file)) {
		return "tool"
	}
	return "product"
}

func generatedFiles(root string, files []string) map[string]bool {
	cmd := exec.Command("git", "-C", root, "check-attr", "-z", "--stdin", "linguist-generated")
	cmd.Stdin = strings.NewReader(strings.Join(files, "\x00") + "\x00")
	out, _ := cmd.Output()
	parts := strings.Split(string(out), "\x00")
	generated := map[string]bool{}
	for i := 0; i+2 < len(parts); i += 3 {
		generated[parts[i]] = contains([]string{"set", "true"}, parts[i+2])
	}
	return generated
}

func (m *Model) inventory(root string, files []string) {
	generated := generatedFiles(root, files)
	claims := map[string]int{}
	for _, file := range files {
		if !sourceFile(file) {
			continue
		}
		source := Source{Path: file, Category: sourceCategory(file), At: Location{file, 1}}
		if generated[file] {
			source.Category = "generated"
		}
		m.ownSource(&source, claims)
		m.Sources = append(m.Sources, source)
		m.goTarget(root, source)
	}
	m.unresolvedClaims(claims)
}

func sourceFile(file string) bool {
	return contains(sourceExtensions, path.Ext(file)) || contains([]string{"project.yml", "project.yaml"}, path.Base(file))
}

func owns(pattern, file string) bool {
	if !strings.HasSuffix(pattern, "/") {
		match, _ := path.Match(pattern, file)
		return match
	}
	for dir := path.Dir(file); dir != "."; dir = path.Dir(dir) {
		if match, _ := path.Match(strings.TrimSuffix(pattern, "/"), dir); match {
			return true
		}
	}
	return false
}

func (m *Model) ownSource(source *Source, claims map[string]int) {
	var owner *Declaration
	for _, d := range m.Declarations {
		for _, pattern := range d.Owns {
			if owns(pattern, source.Path) {
				m.claim(source, &owner, d)
				claims[d.ID+":"+pattern]++
			}
		}
	}
	if owner == nil {
		m.fail(Location{m.Files[0], 1}, "unowned %s source %s", source.Category, source.At)
	}
}

func (m *Model) claim(source *Source, owner **Declaration, d Declaration) {
	if *owner != nil {
		m.fail(d.At, "overlapping ownership of %s; first claimed at %s", source.At, (*owner).At)
	}
	copy := d
	*owner = &copy
	source.Owner = d.ID
}

func (m *Model) unresolvedClaims(claims map[string]int) {
	for _, d := range m.Declarations {
		for _, pattern := range d.Owns {
			if claims[d.ID+":"+pattern] == 0 {
				m.fail(d.At, "ownership claim %q matches no tracked source", pattern)
			}
		}
	}
}

func (m *Model) goTarget(root string, source Source) {
	if !strings.HasSuffix(source.Path, ".go") || source.Category != "product" {
		return
	}
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, source.Path), nil, parser.PackageClauseOnly)
	if err != nil || f.Name.Name != "main" {
		return
	}
	m.addTarget("go:"+path.Dir(source.Path), source)
}

func (m *Model) addTarget(target string, source Source) {
	for _, entry := range m.Sources {
		if entry.Path == target {
			return
		}
	}
	m.Sources = append(m.Sources, Source{Path: target, Category: source.Category, Owner: source.Owner, At: source.At})
}
