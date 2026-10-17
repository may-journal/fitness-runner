package java

import (
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// Plugin is a build tool plugin a check runs once from each build root.
// The repo's build declares it, so the build decides its version and
// config, and a build without it fails with the snippet that adds it.
type Plugin struct {
	// Name labels the plugin in messages.
	Name string
	// Uses holds what the plugin needs from each build tool.
	Uses map[*Build]Use
	// Judge turns a finished run into findings, or none for a clean run.
	Judge func(root string, p Project, o checkkit.Outcome) []string
}

// Use is how one build tool runs a plugin.
type Use struct {
	// Declared matches a build script that applies the plugin.
	Declared *regexp.Regexp
	// Missing is output the tool prints, matched without case, when the
	// build lacks the plugin after all.
	Missing string
	// Snippet is the build file text that adds the plugin.
	Snippet string
	// Args runs the plugin for the project.
	Args func(root string, p Project) []string
}

// Check runs the plugin over the repo at root: a pass with no files in a
// repo without Java, a failure with no JDK, and otherwise one run per build
// root, each failing with its snippet when its build lacks the plugin.
// filesChecked counts the tracked Java sources.
func (pl Plugin) Check(root string) (checkkit.Result, error) {
	roots := Roots(root)
	if len(roots) == 0 {
		return checkkit.Pass(0), nil
	}
	files := len(Sources(root))
	if _, ok := JDK(); !ok {
		return checkkit.Fail(files, NoJDK), nil
	}
	errors, err := pl.runAll(root, roots)
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errors) > 0 {
		return checkkit.Fail(files, errors...), nil
	}
	return checkkit.Pass(files), nil
}

// runAll runs the plugin for each build root and gathers the findings.
func (pl Plugin) runAll(root string, roots []Project) ([]string, error) {
	var errors []string
	for _, p := range roots {
		found, err := pl.run(root, p)
		if err != nil {
			return nil, err
		}
		errors = append(errors, found...)
	}
	return errors, nil
}

// run runs the plugin for one build root and returns its findings.
func (pl Plugin) run(root string, p Project) ([]string, error) {
	use := pl.Uses[p.Build]
	if !Declares(root, p, use.Declared) {
		return []string{pl.missing(p, use)}, nil
	}
	bin, ok := Tool(root, p)
	if !ok {
		return []string{p.Build.Install}, nil
	}
	o := Run(root, bin, p, use.Args(root, p)...)
	if err := o.EnvErr(p.Build.Command); err != nil {
		return nil, err
	}
	if strings.Contains(strings.ToLower(o.Stdout), strings.ToLower(use.Missing)) {
		return []string{pl.missing(p, use)}, nil
	}
	return pl.Judge(root, p, o), nil
}

// missing is the install hint for a build without the plugin.
func (pl Plugin) missing(p Project, use Use) string {
	return pl.Name + " is not in the " + p.Build.Name + " build at " + p.Dir + "; add it: " + use.Snippet
}

// Fallback is the finding for a failed run that named nothing: the tool's
// own reason, else a hint to run the plugin by hand.
func Fallback(o checkkit.Outcome, hint string) []string {
	if o.Code == 0 {
		return nil
	}
	if reasons := ToolErrors(o.Stdout); len(reasons) > 0 {
		return reasons
	}
	return []string{hint}
}
