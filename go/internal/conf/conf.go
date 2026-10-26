// Package conf loads .fitnessrc.json — the Go runner's configuration file.
// Same keys as the TypeScript FitnessConfig; JSON because the standard
// library parses it and a config file must not require a toolchain.
//
//fitness:topic config
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
		// Allow lists exact values the check never flags, for the rare
		// repeat no shared constant can fix. Keep it a shrinking list.
		Allow []string `json:"allow"`
	} `json:"repeatedStringLiterals" check:"repeated-string-literals"`
	// TextReadability holds alarm bands for the text-readability check.
	TextReadability struct {
		// MaxGrade is the Coleman-Liau and ARI grade at which a file alarms.
		MaxGrade float64 `json:"maxGrade"`
		// MaxLix is the LIX score at which a file alarms.
		MaxLix float64 `json:"maxLix"`
		// MinWords is the fewest prose words a file needs to be judged.
		MinWords int `json:"minWords"`
	} `json:"textReadability" check:"text-readability"`
	// ProseBudget holds the prose-budget check's hard caps. Each limit falls
	// back to a built-in default when zero.
	ProseBudget struct {
		// MaxSentenceWords is the most words one sentence may hold.
		MaxSentenceWords int `json:"maxSentenceWords"`
		// MaxParagraphSentences is the most sentences one paragraph may hold.
		MaxParagraphSentences int `json:"maxParagraphSentences"`
		// MaxSectionParagraphs is the most paragraphs one section may hold.
		MaxSectionParagraphs int `json:"maxSectionParagraphs"`
		// MaxListItemWords is the most words one list item may hold.
		MaxListItemWords int `json:"maxListItemWords"`
		// MaxListItems is the most items one list may hold, and the most
		// bullets changelog-bullets allows in a release section.
		MaxListItems int `json:"maxListItems"`
		// MaxWords is the most prose words one section may hold.
		MaxWords int `json:"maxWords"`
	} `json:"proseBudget" check:"prose-budget"`
	// Comment holds comment-structure's budget for an issue or PR comment.
	// Each limit falls back to a built-in default when zero.
	Comment struct {
		// MaxWords is the most words one comment may hold.
		MaxWords int `json:"maxWords"`
		// MaxTables is the most tables one comment may hold.
		MaxTables int `json:"maxTables"`
	} `json:"comment" check:"comment-structure"`
}

// FileName is the config file the runner reads at the repo root.
const FileName = ".fitnessrc.json"

// cspell:ignore Liau

// source is this file, whose doc comments describe each key of Config.
//
//go:embed conf.go
var source []byte

// Defaults holds the value each Config key takes when the config file
// leaves it unset or, for a number, at zero or below.
var Defaults = func() Config {
	var c Config
	c.Policy = "org"
	c.TimeoutMs = 5000
	c.TextReadability.MaxGrade = 18
	c.TextReadability.MaxLix = 60
	c.TextReadability.MinWords = 100
	c.ProseBudget.MaxSentenceWords = 23
	c.ProseBudget.MaxParagraphSentences = 4
	c.ProseBudget.MaxSectionParagraphs = 3
	c.ProseBudget.MaxListItemWords = 23
	c.ProseBudget.MaxListItems = DefaultMaxListItems
	c.ProseBudget.MaxWords = 300
	c.Comment.MaxWords = 100
	c.Comment.MaxTables = 1
	return c
}()

// configSection is Config's registered keys, which Load reads.
var configSection = Register[Config](Defaults, source)

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

// Load reads root's config. A missing file returns (nil, nil): the caller
// falls back to the defaults. A config that still sets a retired exclusion
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
