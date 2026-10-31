package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rolesPackage is the package whose doc comment the 0134 tests edit.
const rolesPackage = "go/internal/goal/goal.go"

// rolesComponent is the component line rolesPackage's doc comment holds.
const rolesComponent = "//fitness:component kit Checks share it as one engine, so no check keeps its own copy.\n"

// rolesRepo copies what the generated check reads from the fitness repo
// these tests live in: its Go code, docs, README, llms page, and the local
// check, with
// rolesPackage's doc comment line old replaced by new.
func rolesRepo(t *testing.T, old, new string) string {
	t.Helper()
	repo := t.TempDir()
	for _, dir := range []string{"go/cmd", "go/internal", "go/check", "docs", "fitness/generated"} {
		mustDo(t, os.CopyFS(filepath.Join(repo, filepath.FromSlash(dir)), os.DirFS(filepath.Join("..", "..", filepath.FromSlash(dir)))))
	}
	for _, file := range []string{"README.md", "llms.txt"} {
		mustDo(t, os.WriteFile(filepath.Join(repo, file), []byte(repoFile(t, file)), 0o644))
	}
	source := repoFile(t, rolesPackage)
	if !strings.Contains(source, old) {
		t.Fatalf("%s has no line %q", rolesPackage, old)
	}
	path := filepath.Join(repo, filepath.FromSlash(rolesPackage))
	mustDo(t, os.WriteFile(path, []byte(strings.Replace(source, old, new, 1)), 0o644))
	return repo
}

// generatedRun runs the candidate `fitness-release generated` in repo.
func generatedRun(t *testing.T, repo string) (string, int) {
	t.Helper()
	return releaseToolRun(t, repo, nil, "generated")
}

func Test0134_1(t *testing.T) {
	t.Parallel()
	out, code := generatedRun(t, rolesRepo(t, rolesComponent, rolesComponent))
	sees(t, out, code, 0, `"ok":true`)
}

func Test0134_2(t *testing.T) {
	t.Parallel()
	out, code := generatedRun(t, rolesRepo(t, rolesComponent, ""))
	sees(t, out, code, 1, "go/internal/goal: its package doc comment needs one", "//fitness:container", "//fitness:component")
}

func Test0134_3(t *testing.T) {
	t.Parallel()
	out, code := generatedRun(t, rolesRepo(t, "//fitness:component kit ", "//fitness:component nowhere "))
	sees(t, out, code, 1, "go/internal/goal: component", "nowhere", "is not a component docs/architecture/03-components.md draws")
}

func Test0134_4(t *testing.T) {
	t.Parallel()
	out, code := generatedRun(t, rolesRepo(t, "//fitness:container cli\n", "//fitness:container nowhere\n"))
	sees(t, out, code, 1, "go/internal/goal: container", "nowhere", "is not a container docs/architecture/02-containers.md draws")
}

func Test0134_5(t *testing.T) {
	t.Parallel()
	out, code := generatedRun(t, rolesRepo(t, rolesComponent, "//fitness:component kit\n"))
	sees(t, out, code, 1, "go/internal/goal: its", "line needs a reason after the ID")
}
