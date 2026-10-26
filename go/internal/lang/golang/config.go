package golang

import (
	_ "embed"

	"github.com/may-journal/fitness-runner/go/internal/conf"
)

// Config is the Go module's own keys in the repo's config file.
type Config struct {
	// TestCoverage declares library entry packages for the optional
	// go-test-coverage check.
	TestCoverage struct {
		// Entries lists the full import paths of library entry packages,
		// which must reach full statement coverage, as command packages do.
		Entries []string `json:"entries"`
	} `json:"goTestCoverage" check:"go-test-coverage"`
	// Complexity holds options for the go-complexity check.
	Complexity struct {
		// Max is the highest cyclomatic complexity one Go function may have.
		Max int `json:"max"`
	} `json:"goComplexity" check:"go-complexity"`
}

// configSource is this file, whose doc comments describe each key of
// Config.
//
//go:embed config.go
var configSource []byte

// Defaults holds the value each Config key takes when the config file
// leaves it unset or at zero.
var Defaults = func() Config {
	var c Config
	c.Complexity.Max = 5
	return c
}()

// section is the Go keys, registered so the schema lists them.
var section = conf.Register[Config](Defaults, configSource)

// LoadConfig reads the Go keys from root's config file. It returns nil,
// nil when the repo has no config file, and fails as conf.Load does.
func LoadConfig(root string) (*Config, error) {
	return section.Load(root)
}
