package vitestconf

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/may-journal/fitness-runner/go/internal/goal"
)

// fullThreshold is the shared coverage goal: every coverage threshold must
// be exactly this value for the vitest-coverage-full gate to pass.
const fullThreshold = goal.Percent

// thresholdKeys are the four coverage threshold keys the TS check demanded.
var thresholdKeys = []string{"branches", "functions", "lines", "statements"}

// thresholdPatterns match `key:\s*100\b` per threshold key against the raw
// config text, comments included.
var thresholdPatterns = buildThresholdPatterns()

func buildThresholdPatterns() []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(thresholdKeys))
	for i, key := range thresholdKeys {
		out[i] = regexp.MustCompile(key + `:\s*` + strconv.Itoa(fullThreshold) + `\b`)
	}
	return out
}

// HasFullThresholds reports whether the Vitest coverage thresholds from root,
// or fallbackRoot when root has no config source, are all exactly 100.
func HasFullThresholds(root, fallbackRoot string) bool {
	if full, decided := fullThresholdsFromRoot(root); decided {
		return full
	}
	if fallbackRoot == "" {
		return false
	}
	full, _ := fullThresholdsFromRoot(fallbackRoot)
	return full
}

// fullThresholdsFromRoot judges one directory, with decided false when it has
// no config source so the caller can fall back.
func fullThresholdsFromRoot(root string) (full, decided bool) {
	sawConfigFile := false
	for _, name := range ConfigNames {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		sawConfigFile = true
		if rawHasFullThresholds(string(data)) {
			return true, true
		}
	}
	if sawConfigFile {
		return false, true
	}
	return fullThresholdsFromPackageJSON(filepath.Join(root, packageJSON))
}

// rawHasFullThresholds is the TS contentHasAllThresholds: every threshold
// key appears as `key: 100` somewhere in the raw text.
func rawHasFullThresholds(content string) bool {
	for _, re := range thresholdPatterns {
		if !re.MatchString(content) {
			return false
		}
	}
	return true
}

// fullThresholdsFromPackageJSON judges the "vitest" object of a package.json,
// full only when all four thresholds are exactly 100.
func fullThresholdsFromPackageJSON(path string) (full, decided bool) {
	vitest, ok := readVitestObject(path)
	if !ok {
		return false, false
	}
	return jsonThresholdsFull(coverageBlockOf(vitest)), true
}

// coverageBlockOf applies the TS getCoverageBlock rule to a decoded vitest
// object: test.coverage when present and non-null, else top-level coverage.
func coverageBlockOf(vitest map[string]any) any {
	if test, ok := vitest["test"].(map[string]any); ok {
		if block, present := test["coverage"]; present && block != nil {
			return block
		}
	}
	return vitest["coverage"]
}

// jsonThresholdsFull reports whether a decoded coverage block carries a
// thresholds object with every threshold key at exactly 100.
func jsonThresholdsFull(block any) bool {
	cov, ok := block.(map[string]any)
	if !ok {
		return false
	}
	thresholds, ok := cov["thresholds"].(map[string]any)
	if !ok {
		return false
	}
	for _, key := range thresholdKeys {
		if !isFullThreshold(thresholds[key]) {
			return false
		}
	}
	return true
}

// isFullThreshold reports whether v is a JSON number equal to the required
// full threshold.
func isFullThreshold(v any) bool {
	value, isNumber := v.(float64)
	return isNumber && value == fullThreshold
}
