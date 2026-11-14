package requirements

import (
	"os/exec"
	"strings"
	"testing"
)

// engineOnly0160 are the engine packages no check may depend on.
var engineOnly0160 = []string{"metadata", "aftereffect", "dupe", "report"}

func Test0160_1(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("go", "list", "-deps", "-test", "-f", "{{.ImportPath}}", "./cmd/fitness-check-...")
	cmd.Dir = ".."
	out, err := cmd.Output()
	mustDo(t, err)
	for dep := range strings.FieldsSeq(string(out)) {
		for _, name := range engineOnly0160 {
			if strings.HasSuffix(dep, "/internal/"+name) {
				t.Errorf("a check depends on %s", dep)
			}
		}
	}
}
