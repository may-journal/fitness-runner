package check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

const (
	manifestPath = ".release-please-manifest.json"
	versionPath  = "version.txt"
)

var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "release-changelog", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	available, err := isAvailable(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	if !available {
		return checkkit.Pass(0), nil
	}
	result, version, ready, err := manifestResult(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	if !ready {
		return result, nil
	}
	return judgeRelease(root, version), nil
}

func isAvailable(root string) (bool, error) {
	_, err := os.Stat(filepath.Join(root, manifestPath))
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func manifestResult(root string) (checkkit.Result, string, bool, error) {
	manifest, err := os.ReadFile(filepath.Join(root, manifestPath))
	if err != nil {
		return checkkit.Result{}, "", false, err
	}

	version, errs := releaseVersion(manifest)
	if len(errs) > 0 {
		return checkkit.Fail(1, errs...), "", false, nil
	}
	if version == "" {
		return checkkit.Pass(1), "", false, nil
	}
	return checkkit.Result{}, version, true, nil
}

// judgeRelease checks the manifest and version.txt agree on version.
func judgeRelease(root, version string) checkkit.Result {
	if errs := versionFileErrors(root, version); len(errs) > 0 {
		return checkkit.Fail(2, errs...)
	}
	return checkkit.Pass(2)
}

func releaseVersion(raw []byte) (string, []string) {
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", []string{manifestPath + " is invalid JSON"}
	}
	if manifest == nil {
		return "", []string{manifestPath + " must be a JSON object"}
	}
	if len(manifest) == 0 {
		return "", nil
	}
	value, exists := manifest["."]
	if !exists {
		return "", []string{manifestPath + ` must contain a root "." release version`}
	}
	return validRootVersion(value)
}

func validRootVersion(value any) (string, []string) {
	version, ok := value.(string)
	if !ok {
		return "", []string{fmt.Sprintf("%s root version must be numeric major.minor.patch (got %q)", manifestPath, value)}
	}
	if !semverPattern.MatchString(version) {
		return "", []string{fmt.Sprintf("%s root version must be numeric major.minor.patch (got %q)", manifestPath, value)}
	}
	return version, nil
}

// versionFileErrors names a missing version.txt, or one whose version
// differs from the manifest's.
func versionFileErrors(root, version string) []string {
	versionText, err := os.ReadFile(filepath.Join(root, versionPath))
	if err != nil {
		return []string{versionPath + " is missing; it must match release " + version}
	}
	if got := strings.TrimSpace(string(versionText)); got != version {
		return []string{fmt.Sprintf("%s version %q must match %s version %q", versionPath, got, manifestPath, version)}
	}
	return nil
}
