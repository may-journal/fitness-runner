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
		if !os.IsNotExist(err) {
			return nil, err
		}
		for _, name := range legacyNames {
			if _, statErr := os.Stat(filepath.Join(root, name)); statErr == nil {
				return nil, ErrLegacyConfig
			}
		}
		return nil, nil
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
