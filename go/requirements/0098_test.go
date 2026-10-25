package requirements

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func Test0098_1(t *testing.T) {
	t.Parallel()
	folder := filepath.Join(t.TempDir(), "skills")
	out, code := fiRun(t, t.TempDir(), nil, "--link-skills", folder)
	sees(t, out, code, 0, "fitness: skill "+filepath.Join(folder, "add-requirements")+" linked to ")
	sees(t, fiFile(t, filepath.Join(folder, "add-requirements", "SKILL.md")), 0, 0, "name: add-requirements")
}

func Test0098_2(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	mustDo(t, os.Symlink(filepath.Join(folder, "gone", "skills", "add-requirements"), filepath.Join(folder, "add-requirements")))
	write(t, folder, map[string]string{"mine/SKILL.md": "mine\n"})
	out, code := fiRun(t, t.TempDir(), nil, "--link-skills", folder)
	sees(t, out, code, 0, "add-requirements linked to ")
	sees(t, fiFile(t, filepath.Join(folder, "add-requirements", "SKILL.md")), 0, 0, "name: add-requirements")
	sees(t, fiFile(t, filepath.Join(folder, "mine", "SKILL.md")), 0, 0, "mine\n")
}

func Test0098_3(t *testing.T) {
	t.Parallel()
	folder := filepath.Join(t.TempDir(), "skills")
	out, code := fiRun(t, t.TempDir(), nil, "--install-hook", "--link-skills", folder)
	sees(t, out, code, 1, "link-skills does not combine with other modes")
	if _, err := os.Lstat(folder); err == nil {
		t.Errorf("skills folder %s was created", folder)
	}
}

func Test0098_4(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	fiRun(t, t.TempDir(), nil, "--link-skills", folder)
	bundle := filepath.Dir(mustLink(t, filepath.Join(folder, "add-requirements")))
	mustDo(t, os.Symlink(filepath.Join(bundle, "retired"), filepath.Join(folder, "retired")))
	other := filepath.Join(t.TempDir(), "bundle-1", "skills", "other")
	write(t, other, map[string]string{"SKILL.md": "other\n"})
	mustDo(t, os.Symlink(other, filepath.Join(folder, "foreign")))
	out, code := fiRun(t, t.TempDir(), nil, "--link-skills", folder)
	sees(t, out, code, 0, "retired removed")
	if _, err := os.Lstat(filepath.Join(folder, "retired")); err == nil {
		t.Errorf("link to the dropped skill stayed")
	}
	sees(t, fiFile(t, filepath.Join(folder, "foreign", "SKILL.md")), 0, 0, "other\n")
}

func Test0098_5(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	fiRun(t, t.TempDir(), nil, "--link-skills", folder)
	for _, name := range []string{"add-requirements", filepath.Join("add-requirements", "SKILL.md")} {
		info, err := os.Stat(filepath.Join(folder, name))
		mustDo(t, err)
		if info.Mode().Perm()&0o044 == 0 {
			t.Errorf("%s mode %v is not readable by others", name, info.Mode().Perm())
		}
	}
}

func mustLink(t *testing.T, path string) string {
	t.Helper()
	target, err := os.Readlink(path)
	mustDo(t, err)
	return target
}

// skillLink matches each Markdown link target in a skill.
var skillLink = regexp.MustCompile(`\]\(([^)\s]+)\)`)

func Test0098_6(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	out, code := fiRun(t, t.TempDir(), nil, "--link-skills", folder)
	sees(t, out, code, 0, "fitness: skill "+filepath.Join(folder, "fitness")+" linked to ")
	skill := fiFile(t, filepath.Join(folder, "fitness", "SKILL.md"))
	sees(t, skill, 0, 0, "name: fitness\n", "](../../docs/checks.md)", "](../../docs/requirements)", "](../../docs/architecture)")
	source := filepath.Join("..", "..", "skills", "fitness")
	if kept := fiFile(t, filepath.Join(source, "SKILL.md")); kept != skill {
		t.Fatalf("linked fitness skill differs from the one kept in %s", source)
	}
	for _, link := range skillLink.FindAllStringSubmatch(skill, -1) {
		resolves(t, filepath.Join(source, link[1]), "fitness skill link "+link[1])
	}
	for _, name := range checkNames(t) {
		resolves(t, filepath.Join("..", "cmd", "fitness-check-"+name, "README.md"), "README the fitness skill names for "+name)
	}
}

// resolves fails the test when what a skill points the reader at is missing.
func resolves(t *testing.T, path, what string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("%s does not resolve: %v", what, err)
	}
}

func Test0098_7(t *testing.T) {
	t.Parallel()
	folder := t.TempDir()
	fiRun(t, t.TempDir(), nil, "--link-skills", folder)
	linked, err := os.ReadDir(folder)
	mustDo(t, err)
	if len(linked) == 0 {
		t.Fatal("no skill was linked")
	}
	checks := map[string]bool{}
	for _, name := range checkNames(t) {
		checks[name] = true
	}
	for _, skill := range linked {
		if checks[skill.Name()] {
			t.Errorf("skill %s is named for a single check", skill.Name())
		}
	}
}

// checkNames lists every check by the name a user passes to fitness.
func checkNames(t *testing.T) []string {
	t.Helper()
	folders, err := filepath.Glob(filepath.Join("..", "cmd", "fitness-check-*"))
	mustDo(t, err)
	if len(folders) == 0 {
		t.Fatal("no check folders found under go/cmd")
	}
	names := make([]string, 0, len(folders))
	for _, folder := range folders {
		names = append(names, strings.TrimPrefix(filepath.Base(folder), "fitness-check-"))
	}
	return names
}
