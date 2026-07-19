// Package conf loads .fitnessrc.json — the Go runner's configuration file.
// Same keys as the TypeScript FitnessConfig; JSON because the standard
// library parses it and a config file must not require a toolchain.
package conf

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config is the shape of .fitnessrc.json.
type Config struct {
	// Checks is the ordered list of check names and/or local executable
	// paths (entries containing a path separator). When set, only these run.
	Checks []string `json:"checks"`
	// DisabledChecks removes name entries from the resolved list; path
	// entries are opt-in only and never removed.
	DisabledChecks []string `json:"disabledChecks"`
	// SkipTheseDirectories overrides the walker's skip-dir set.
	SkipTheseDirectories []string `json:"skipTheseDirectories"`
	// RepeatedStringLiterals holds options for that check.
	RepeatedStringLiterals struct {
		Allow []string `json:"allow"`
	} `json:"repeatedStringLiterals"`
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
// config returns ErrLegacyConfig.
func Load(root string) (*Config, error) {
	raw, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		return nil, missingConfigErr(root, err)
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
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
