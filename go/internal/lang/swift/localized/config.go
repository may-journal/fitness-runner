package localized

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/conf"
)

// fileConfig is the check's own key in the repo's config file.
type fileConfig struct {
	// LocalizedText holds the localized-text check's options.
	LocalizedText struct {
		// Reference names the type views and tests read catalog entries
		// through, such as Words.
		Reference string `json:"reference"`
		// Allow lists literals the check passes, each with its reason.
		Allow []struct {
			// Pattern is a regular expression a literal must match whole.
			Pattern string `json:"pattern"`
			// Reason says why the literal stays out of the catalog.
			Reason string `json:"reason"`
		} `json:"allow"`
	} `json:"localizedText" check:"localized-text"`
}

// config is the loaded check config: how a failure names the reference,
// and the compiled allow patterns.
type config struct {
	reference string
	allow     []*regexp.Regexp
}

// defaultReference is what a failure says to use when the repo names no
// reference type.
const defaultReference = "String(localized:)"

// loadConfig reads localizedText from root's config file. A pattern with
// no reason, or one that is no regular expression, is a problem the check
// fails with.
func loadConfig(root string) (config, []string, error) {
	file, err := conf.LoadAs[fileConfig](root)
	if err != nil || file == nil {
		return config{reference: defaultReference}, nil, err
	}
	cfg := config{reference: reference(file.LocalizedText.Reference)}
	var problems []string
	for _, a := range file.LocalizedText.Allow {
		re, problem := compileAllow(a.Pattern, a.Reason)
		if problem != "" {
			problems = append(problems, problem)
			continue
		}
		cfg.allow = append(cfg.allow, re)
	}
	return cfg, problems, nil
}

// reference is how a failure names the configured reference type, or
// the default when none is named.
func reference(name string) string {
	if name = strings.TrimSpace(name); name != "" {
		return "a " + name + " reference"
	}
	return defaultReference
}

// compileAllow compiles one allow pattern to match a whole literal, or
// returns the problem with it.
func compileAllow(pattern, reason string) (*regexp.Regexp, string) {
	re, err := regexp.Compile(`^(?:` + pattern + `)$`)
	if err != nil {
		return nil, fmt.Sprintf("localizedText.allow pattern %q is not a regular expression: %v", pattern, err)
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Sprintf("localizedText.allow pattern %q needs a reason", pattern)
	}
	return re, ""
}

// allowed reports whether an allow pattern matches the whole text.
func (c config) allowed(text string) bool {
	for _, re := range c.allow {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}
