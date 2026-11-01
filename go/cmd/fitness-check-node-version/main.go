package check

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/node"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "node-version", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsNotApplicable},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	if !node.Applies(root) {
		return checkkit.Pass(0), nil
	}
	raw, err := os.ReadFile(filepath.Join(root, ".nvmrc"))
	if err != nil {
		return checkkit.Fail(1, "missing .nvmrc"), nil
	}
	current, err := currentNodeVersion()
	if err != nil {
		return checkkit.Result{}, err
	}
	return judge(strings.TrimSpace(string(raw)), current), nil
}

// judge passes when the current major version is at least the .nvmrc requirement, reporting "NaN" for an unparsable one.
func judge(required, current string) checkkit.Result {
	requiredMajor, requiredOk := versionMajor(required)
	currentMajor, currentOk := versionMajor(current)
	if requiredOk && currentOk && currentMajor >= requiredMajor {
		return checkkit.Pass(1)
	}
	requiredText := "NaN"
	if requiredOk {
		requiredText = fmt.Sprintf("%d", requiredMajor)
	}
	return checkkit.Fail(1, fmt.Sprintf(
		"Node %s does not satisfy .nvmrc (requires %s.x). Run: nvm use", current, requiredText))
}

var versionRe = regexp.MustCompile(`^v?(\d+)`)

// versionMajor parses the leading major version from e.g. "v24.0.0" or "24".
func versionMajor(version string) (int, bool) {
	m := versionRe.FindStringSubmatch(strings.TrimSpace(version))
	if m == nil {
		return 0, false
	}
	var major int
	if _, err := fmt.Sscanf(m[1], "%d", &major); err != nil {
		return 0, false
	}
	return major, true
}

// nodeMissing is the info a run without node on PATH reports.
const nodeMissing = "node not found on PATH. Install Node.js (e.g. via nvm) to run the node-version check"

// currentNodeVersion runs `node --version`. A missing, crashed or killed node is an internal error, since no version was read.
func currentNodeVersion() (string, error) {
	bin, err := exec.LookPath("node")
	if err != nil {
		return "", checkkit.Internal(errors.New(nodeMissing))
	}
	o := checkkit.Exec(exec.Command(bin, "--version"))
	if err := o.EnvErr("node --version"); err != nil {
		return "", err
	}
	if o.Code != 0 {
		return "", checkkit.NoVerdict("node --version", o.Code, o.Stderr+"\n"+o.Stdout)
	}
	return strings.TrimSpace(o.Stdout), nil
}
