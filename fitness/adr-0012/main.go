// Command adr-0012 keeps ADR 0012: fitness keeps each of its own checks in
// a fitness/<name>/ folder, found with no config, and no flat check program
// anywhere else.
package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// rule is the check, kept apart from main so a Go test could run it too.
var rule = check.Check{Run: run}

func main() { check.Main(rule) }

// run judges the config and every tracked file's place.
func run(root string, _ []string) (check.Result, error) {
	config, err := os.ReadFile(filepath.Join(root, ".fitnessrc.json"))
	if err != nil {
		return check.Result{}, err
	}
	var errs []string
	if strings.Contains(string(config), `"localChecks"`) {
		errs = append(errs, ".fitnessrc.json: fitness finds its checks in fitness/ with no config, so drop localChecks (ADR 0012, decision 1)")
	}
	files := check.Files(root)
	for _, f := range files {
		errs = append(errs, placed(f)...)
	}
	if len(errs) > 0 {
		return check.Fail(len(files)+1, errs...), nil
	}
	return check.Pass(len(files) + 1), nil
}

// placed fails a flat check program outside go/cmd, which belongs in a
// fitness/<name>/ folder with its README.
func placed(f string) []string {
	if !strings.HasPrefix(path.Base(f), "fitness-check-") || strings.HasPrefix(f, "go/cmd/") {
		return nil
	}
	return []string{f + ": move this check into fitness/<name>/ with a README.md (ADR 0012, decisions 1 and 2)"}
}
