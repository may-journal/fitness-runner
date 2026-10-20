package requirements

import (
	"os"
	"path/filepath"
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
