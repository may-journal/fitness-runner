// Package conf loads the repository's .fitnessrc.json, giving each check a typed section for its own keys (go/docs/config.md).
package conf

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is the shape of .fitnessrc.json, the optional file at a repo's
// root that tunes which checks run and how strictly they judge.
type Config struct {
	// Policy selects org or external use; both default to all checks.
	// External lets a repo pick, replace, and turn off checks; org runs every
	// check as the shared bar.
	Policy string `json:"policy" enum:"org,external"`
	// LegacyChecks optionally selects checks in external mode, which org ignores.
	LegacyChecks []string `json:"checks"`
	// LocalChecks names a repo directory of the repo's own check programs,
	// searched before the bundled ones, as ADR 0009 describes.
	LocalChecks string `json:"localChecks"`
	// DisabledChecks names checks external policy leaves out and org policy
	// refuses, as ADR 0009 describes.
	DisabledChecks []string `json:"disabledChecks"`
	// RequiredChecks names checks whose internal error fails the run instead
	// of showing info, under either policy; `--require` adds to them.
	RequiredChecks []string `json:"requiredChecks"`
	// CheckVersions pins each check the repo uses to a release such as v1.2.3
	// or a major such as v1; a "*" key pins all. Unset or empty, the repo uses
	// every check at its current major.
	CheckVersions map[string]string `json:"checkVersions"`
	// PlanApprovers lists the GitHub logins whose comment saying approve
	// approves a Plan for the pre-push gate, under either policy. Unset, the
	// repo's owner and its org's members may approve, as GitHub reports each
	// comment author's association.
	PlanApprovers []string `json:"planApprovers"`
	// ScopedPullRequests true skips a pull request's full scan, for a huge
	// repo. A push to main still scans every file.
	ScopedPullRequests bool `json:"scopedPullRequests"`
}

// FileName is the config file the runner reads at the repo root.
const FileName = ".fitnessrc.json"

// source is this file, whose doc comments describe each key of Config.
//
//go:embed conf.go
var source []byte

// Defaults holds the value each Config key takes when the config file
// leaves it unset.
var Defaults = func() Config {
	var c Config
	c.Policy = "org"
	return c
}()

// configSection is Config's registered keys, which Load reads.
var configSection = Register[Config](Defaults, source)

// Load reads root's config, and a missing file returns (nil, nil) so the
// caller falls back to defaults. A config that still sets a retired exclusion
// key returns an error naming each one.
func Load(root string) (*Config, error) {
	return configSection.Load(root)
}

// Load reads root's config into a T, which holds the keys a caller owns,
// such as a language module's options. It returns (nil, nil) without a
// config file and fails as the package's Load does.
func (*Section[T]) Load(root string) (*T, error) {
	raw, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		return nil, ignoreMissing(err)
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	if retired := retiredKeys(raw); len(retired) > 0 {
		return nil, fmt.Errorf("%s sets %s; every check now judges every tracked file, so remove %s and fix the findings instead",
			FileName, strings.Join(retired, ", "), pronoun(len(retired)))
	}
	return &v, nil
}

// ignoreMissing is err, or nil when it says the config file does not exist.
func ignoreMissing(err error) error {
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// retiredKeys names the exclusion keys raw still sets, in a fixed order:
// top-level ignore and skipTheseDirectories, then proseBudget.exempt.
func retiredKeys(raw []byte) []string {
	var top struct {
		Ignore      json.RawMessage `json:"ignore"`
		SkipDirs    json.RawMessage `json:"skipTheseDirectories"`
		ProseBudget struct {
			Exempt json.RawMessage `json:"exempt"`
		} `json:"proseBudget"`
	}
	_ = json.Unmarshal(raw, &top) // raw already parsed as a Config
	var names []string
	for _, k := range []struct {
		name string
		val  json.RawMessage
	}{
		{"ignore", top.Ignore},
		{"skipTheseDirectories", top.SkipDirs},
		{"proseBudget.exempt", top.ProseBudget.Exempt},
	} {
		if k.val != nil {
			names = append(names, `"`+k.name+`"`)
		}
	}
	return names
}

// pronoun picks "it" or "them" for the retired-key error.
func pronoun(n int) string {
	if n == 1 {
		return "it"
	}
	return "them"
}
