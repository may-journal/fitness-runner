package golang

import "github.com/may-journal/fitness-runner/go/internal/conf"

// Config is the Go module's own keys in the repo's config file.
type Config struct {
	// TestCoverage declares library entry packages for the optional
	// go-test-coverage check.
	TestCoverage struct {
		Entries []string `json:"entries"`
	} `json:"goTestCoverage"`
	// Complexity holds options for the go-complexity check.
	Complexity struct {
		Max int `json:"max"`
	} `json:"goComplexity"`
}

// LoadConfig reads the Go keys from root's config file. It returns nil,
// nil when the repo has no config file, and fails as conf.Load does.
func LoadConfig(root string) (*Config, error) {
	return conf.LoadAs[Config](root)
}
