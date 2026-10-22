package golang

import "github.com/may-journal/fitness-runner/go/internal/conf"

// Config is the Go module's own keys in the repo's config file.
type Config struct {
	// TestCoverage declares library entry packages for the optional
	// go-test-coverage check.
	TestCoverage struct {
		// Entries lists library entry packages by full import path. Each
		// must reach full coverage, as every command package must.
		Entries []string `json:"entries"`
	} `json:"goTestCoverage" check:"go-test-coverage"`
	// Complexity holds options for the go-complexity check.
	Complexity struct {
		// Max is the highest cyclomatic complexity a function may have; 5
		// when zero.
		Max int `json:"max"`
	} `json:"goComplexity" check:"go-complexity"`
}

// LoadConfig reads the Go keys from root's config file. It returns nil,
// nil when the repo has no config file, and fails as conf.Load does.
func LoadConfig(root string) (*Config, error) {
	return conf.LoadAs[Config](root)
}
