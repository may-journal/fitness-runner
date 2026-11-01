package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/node"
	"github.com/may-journal/fitness-runner/go/internal/lang/node/vitestconf"
	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "vitest-coverage-exclude", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsStripped},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	if !node.Applies(root) {
		return checkkit.Pass(0), nil
	}
	return judge(loadExclude(root)), nil
}

// loadExclude resolves the coverage exclude list like the TS check did, from root when it has a config, else from sharedconf.ResolveDir.
func loadExclude(root string) []string {
	if exclude, found := vitestconf.LoadExcludeFromRoot(root); found {
		return exclude
	}
	exclude, _ := vitestconf.LoadExcludeFromRoot(sharedconf.ResolveDir(root))
	return exclude
}

// judge fails every coverage exclude entry; filesChecked is always 1 — the
// one config source — exactly like the TS check.
func judge(exclude []string) checkkit.Result {
	var errs []string
	for _, pattern := range exclude {
		errs = append(errs, fmt.Sprintf(
			"Vitest coverage exclude must be empty, so coverage judges every file; remove: %s",
			jsonQuote(pattern)))
	}
	if len(errs) == 0 {
		return checkkit.Pass(1)
	}
	return checkkit.Fail(1, errs...)
}

// jsonQuote renders a pattern exactly as the TS JSON.stringify did —
// double-quoted, without Go's default HTML escaping.
func jsonQuote(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(buf.String(), "\n")
}
