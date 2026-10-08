// Package conf loads .fitnessrc.json — the Go runner's configuration file.
// Same keys as the TypeScript FitnessConfig; JSON because the standard
// library parses it and a config file must not require a toolchain.
package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is the shape of .fitnessrc.json.
type Config struct {
	// Policy selects org or external use; both default to all checks.
	Policy string `json:"policy"`
	// LegacyChecks optionally selects checks in external mode. Org ignores it.
	LegacyChecks []string `json:"checks"`
	// TimeoutMs replaces the default per-check budget for checks that declare
	// none of their own.
	TimeoutMs int `json:"timeoutMs"`
	// RepeatedStringLiterals holds options for that check.
	RepeatedStringLiterals struct {
		Allow []string `json:"allow"`
	} `json:"repeatedStringLiterals"`
	// GoTestCoverage declares library entry packages for the optional coverage check.
	GoTestCoverage struct {
		Entries []string `json:"entries"`
	} `json:"goTestCoverage"`
	// GoComplexity holds options for the go-complexity check.
	GoComplexity struct {
		Max int `json:"max"`
	} `json:"goComplexity"`
	// TextReadability holds alarm bands for the text-readability check.
	TextReadability struct {
		MaxGrade float64 `json:"maxGrade"`
		MaxLix   float64 `json:"maxLix"`
		MinWords int     `json:"minWords"`
	} `json:"textReadability"`
	// ProseBudget holds the prose-budget check's hard caps. Each limit falls
	// back to a built-in default when zero.
	ProseBudget struct {
		MaxSentenceWords      int `json:"maxSentenceWords"`
		MaxParagraphSentences int `json:"maxParagraphSentences"`
		MaxSectionParagraphs  int `json:"maxSectionParagraphs"`
		MaxListItemWords      int `json:"maxListItemWords"`
		MaxListItems          int `json:"maxListItems"`
		MaxWords              int `json:"maxWords"`
	} `json:"proseBudget"`
}

// FileName is the config file the runner reads at the repo root.
const FileName = ".fitnessrc.json"

// legacyNames are configs the Go runner cannot evaluate; finding one (with
// no .fitnessrc.json beside it) earns a migration hint.
var legacyNames = []string{".fitnessrc.ts", ".fitnessrc.js", ".fitnessrc.cjs", ".fitnessrc.mjs"}

// ErrLegacyConfig means only a JS/TS config exists at the root.
var ErrLegacyConfig = errors.New(
	"JS/TS fitness config is not supported by the Go runner; migrate to " + FileName)

// Load reads root's config. A missing file returns (nil, nil): the caller
// falls back to the default check list. A legacy JS/TS config with no JSON
// config returns ErrLegacyConfig. A config that still sets a retired
// exclusion key returns an error naming each one.
func Load(root string) (*Config, error) {
	raw, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		return nil, missingConfigErr(root, err)
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if retired := retiredKeys(raw); len(retired) > 0 {
		return nil, fmt.Errorf("%s sets %s; every check now judges every tracked file, so remove %s and fix the findings instead",
			FileName, strings.Join(retired, ", "), pronoun(len(retired)))
	}
	return &c, nil
}

// retiredKeys names the exclusion keys raw still sets, in a fixed order:
// top-level ignore, skipTheseDirectories, and disabledChecks, then
// proseBudget.exempt.
func retiredKeys(raw []byte) []string {
	var top struct {
		Ignore      json.RawMessage `json:"ignore"`
		SkipDirs    json.RawMessage `json:"skipTheseDirectories"`
		Disabled    json.RawMessage `json:"disabledChecks"`
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
		{"disabledChecks", top.Disabled},
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

// missingConfigErr maps a failed .fitnessrc.json read to Load's error: a
// non-not-exist error passes through, a legacy JS/TS config beside the
// missing file earns ErrLegacyConfig, and a plainly absent config is nil
// (the caller falls back to defaults).
func missingConfigErr(root string, readErr error) error {
	if !os.IsNotExist(readErr) {
		return readErr
	}
	for _, name := range legacyNames {
		if _, statErr := os.Stat(filepath.Join(root, name)); statErr == nil {
			return ErrLegacyConfig
		}
	}
	return nil
}
