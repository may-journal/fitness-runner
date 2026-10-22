// cspell:ignore Liau

// Package conf loads .fitnessrc.json — the Go runner's configuration file.
// Same keys as the TypeScript FitnessConfig; JSON because the standard
// library parses it and a config file must not require a toolchain.
//
//fitness:topic config
package conf

import (
	"encoding/json"
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
	// LocalChecks names a repo directory of the repo's own check programs,
	// searched before the bundled ones. See ADR 0009.
	LocalChecks string `json:"localChecks"`
	// DisabledChecks names checks external policy leaves out; org policy
	// refuses it. See ADR 0009.
	DisabledChecks []string `json:"disabledChecks"`
	// RepeatedStringLiterals holds options for that check.
	RepeatedStringLiterals struct {
		// Allow lists exact values never flagged, for the rare repeat no
		// shared constant can fix. Keep it a shrinking baseline.
		Allow []string `json:"allow"`
	} `json:"repeatedStringLiterals" check:"repeated-string-literals"`
	// TextReadability holds alarm bands for the text-readability check.
	TextReadability struct {
		// MaxGrade is the Coleman-Liau and ARI grade at which a score is in
		// alarm territory; 18 when zero.
		MaxGrade float64 `json:"maxGrade"`
		// MaxLix is the LIX score at which a score is in alarm territory; 60
		// when zero.
		MaxLix float64 `json:"maxLix"`
		// MinWords is the fewest prose words a file needs to be judged; 100
		// when zero.
		MinWords int `json:"minWords"`
	} `json:"textReadability" check:"text-readability"`
	// ProseBudget holds the prose-budget check's hard caps. Each limit falls
	// back to a built-in default when zero.
	ProseBudget struct {
		// MaxSentenceWords caps words per sentence; 23 by default.
		MaxSentenceWords int `json:"maxSentenceWords"`
		// MaxParagraphSentences caps sentences per paragraph; 4 by default.
		MaxParagraphSentences int `json:"maxParagraphSentences"`
		// MaxSectionParagraphs caps paragraphs per section; 3 by default.
		MaxSectionParagraphs int `json:"maxSectionParagraphs"`
		// MaxListItemWords caps words per list item; 23 by default.
		MaxListItemWords int `json:"maxListItemWords"`
		// MaxListItems caps items per list; 8 by default.
		MaxListItems int `json:"maxListItems"`
		// MaxWords caps prose words per section; 300 by default.
		MaxWords int `json:"maxWords"`
	} `json:"proseBudget" check:"prose-budget"`
	// Comment holds comment-structure's budget for an issue or PR comment.
	// Each limit falls back to a built-in default when zero.
	Comment struct {
		// MaxWords caps words outside tables, code, and diagrams; 100 by
		// default.
		MaxWords int `json:"maxWords"`
		// MaxTables caps plain tables; 1 by default.
		MaxTables int `json:"maxTables"`
	} `json:"comment" check:"comment-structure"`
}

// FileName is the config file the runner reads at the repo root.
const FileName = ".fitnessrc.json"

// Load reads root's config. A missing file returns (nil, nil): the caller
// falls back to the defaults. A config that still sets a retired exclusion
// key returns an error naming each one.
// DefaultMaxListItems is the longest list prose-budget allows, and the most
// bullets changelog-bullets allows in a release section, unless
// proseBudget.maxListItems sets another limit.
const DefaultMaxListItems = 8

// MaxListItems is the repo's list limit: proseBudget.maxListItems when set,
// else DefaultMaxListItems.
func MaxListItems(root string) int {
	if cfg, err := Load(root); err == nil && cfg != nil && cfg.ProseBudget.MaxListItems > 0 {
		return cfg.ProseBudget.MaxListItems
	}
	return DefaultMaxListItems
}

func Load(root string) (*Config, error) {
	return LoadAs[Config](root)
}

// LoadAs reads root's config into a T, which holds the keys a caller owns,
// such as a language module's options. It returns (nil, nil) without a
// config file and fails as Load does.
func LoadAs[T any](root string) (*T, error) {
	raw, err := os.ReadFile(filepath.Join(root, FileName))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
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
