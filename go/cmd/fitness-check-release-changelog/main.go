// Command fitness-check-release-changelog binds Release Please's manifest to
// the version file and the human-readable release section in CHANGELOG.md.
package main

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
	manifestPath  = ".release-please-manifest.json"
	versionPath   = "version.txt"
	changelogPath = "CHANGELOG.md"
)

var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "release-changelog"},
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
	return judgeRelease(root, version, checkkit.ChangedFiles()), nil
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

func judgeRelease(root, version string, changed []string) checkkit.Result {
	filesChecked := 3
	errs := correlatedFileErrors(root, version)
	if manifestChanged(changed) && !contains(changed, changelogPath) {
		errs = append(errs, manifestPath+" changed without "+changelogPath+"; add the release section reviewers will approve")
	}
	if len(errs) > 0 {
		return checkkit.Fail(filesChecked, errs...)
	}
	return checkkit.Pass(filesChecked)
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

func correlatedFileErrors(root, version string) []string {
	var errs []string
	versionText, err := os.ReadFile(filepath.Join(root, versionPath))
	if err != nil {
		errs = append(errs, versionPath+" is missing; it must match release "+version)
	} else if got := strings.TrimSpace(string(versionText)); got != version {
		errs = append(errs, fmt.Sprintf("%s version %q must match %s version %q", versionPath, got, manifestPath, version))
	}

	changelog, err := os.ReadFile(filepath.Join(root, changelogPath))
	if err != nil {
		return append(errs, changelogPath+" is missing; add a release section for "+version)
	}
	return append(errs, releaseSectionErrors(string(changelog), version)...)
}

func releaseSectionErrors(changelog, version string) []string {
	sections := releaseSections(changelog, version)
	if len(sections) == 0 {
		return []string{changelogPath + " must contain a `## " + version + "` release section"}
	}
	if len(sections) > 1 {
		return []string{changelogPath + " contains duplicate release sections for " + version}
	}
	if hasListItem(sections[0]) {
		return nil
	}
	return []string{changelogPath + " release section " + version + " must contain at least one list item describing what ships"}
}

func hasListItem(section string) bool {
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			return true
		}
		if strings.HasPrefix(trimmed, "* ") {
			return true
		}
	}
	return false
}

func releaseSections(changelog, version string) []string {
	lines := strings.Split(changelog, "\n")
	heading := regexp.MustCompile(`^##\s+\[?` + regexp.QuoteMeta(version) + `\]?(?:\s|$)`)
	var sections []string
	for i := 0; i < len(lines); i++ {
		if !heading.MatchString(lines[i]) {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(lines[j], "## ") {
				end = j
				break
			}
		}
		sections = append(sections, strings.Join(lines[i:end], "\n"))
	}
	return sections
}

func manifestChanged(files []string) bool {
	return contains(files, manifestPath)
}

func contains(files []string, target string) bool {
	for _, file := range files {
		if filepath.ToSlash(file) == target {
			return true
		}
	}
	return false
}
