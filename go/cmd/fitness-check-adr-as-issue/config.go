package check

import (
	_ "embed"
	"fmt"
	"path"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/conf"
)

// fileConfig is the check's own key in the repo's config file.
type fileConfig struct {
	// AdrAsIssue holds the adr-as-issue check's options.
	AdrAsIssue struct {
		// Allow lists ADR files still waiting to move into issues, each
		// with its reason. It is a fallback while `fitness adr-migrate`
		// cannot run; keep it a shrinking list.
		Allow []struct {
			// Path is a glob, as Go's path.Match reads it, of the waiting files.
			Path string `json:"path"`
			// Reason says why the files have not moved yet.
			Reason string `json:"reason"`
		} `json:"allow"`
	} `json:"adrAsIssue" check:"adr-as-issue"`
}

// configSource is this file, whose doc comments describe each adrAsIssue key.
//
//go:embed config.go
var configSource []byte

// section is the adrAsIssue keys, registered so the schema lists them.
var section = conf.Register[fileConfig](fileConfig{}, configSource)

// config is the loaded allow globs, each with a reason.
type config struct {
	allow []string
}

// loadConfig reads adrAsIssue from root's config file. Each allow entry
// that lacks a reason or is no glob is a problem the check fails with.
func loadConfig(root string) (config, []string, error) {
	file, err := section.Load(root)
	if err != nil || file == nil {
		return config{}, nil, err
	}
	var cfg config
	var problems []string
	for _, a := range file.AdrAsIssue.Allow {
		if problem := allowProblem(a.Path, a.Reason); problem != "" {
			problems = append(problems, problem)
			continue
		}
		cfg.allow = append(cfg.allow, a.Path)
	}
	return cfg, problems, nil
}

// allowProblem is what is wrong with one allow entry, or "".
func allowProblem(glob, reason string) string {
	if _, err := path.Match(glob, ""); err != nil {
		return fmt.Sprintf("adrAsIssue.allow path %q is not a glob: %v", glob, err)
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Sprintf("adrAsIssue.allow path %q needs a reason", glob)
	}
	return ""
}

// allowedBy is the first allow glob that matches rel, or "".
func (c config) allowedBy(rel string) string {
	for _, glob := range c.allow {
		if ok, _ := path.Match(glob, rel); ok {
			return glob
		}
	}
	return ""
}
