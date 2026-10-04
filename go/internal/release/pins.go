package release

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
)

var ErrPinDowngrade = errors.New("release pins are newer")

const pinVersion = `(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)`

var pinVersionPattern = regexp.MustCompile(`^` + pinVersion + `$`)
var pinNumber = regexp.MustCompile(pinVersion)
var pinTag = regexp.MustCompile(`(?:go/)?v` + pinVersion)

// PinFiles lists the release references maintained by the release workflow.
func PinFiles() []string {
	return []string{
		"action.yml", "go/cmd/fitness-install/main.go",
		".github/workflows/ci-reusable.yml", ".github/workflows/plan-check-reusable.yml",
		".github/workflows/pr-check-reusable.yml", ".github/workflows/close-check-reusable.yml",
		"docs/ci.md", "README.md",
	}
}

func referencePattern(path string) *regexp.Regexp {
	switch path {
	case "go/cmd/fitness-install/main.go":
		return regexp.MustCompile(`var version = "` + pinVersion + `"`)
	case "README.md":
		return regexp.MustCompile("Pin a version with `@(?:go/)?v" + pinVersion + "`")
	case "docs/ci.md", "action.yml":
		return regexp.MustCompile(`(?:https://github\.com/may-journal/fitness-runner/releases/download/(?:go/)?v|may-journal/fitness-runner@(?:go/)?v|fitness-install-|` + "`(?:go/)?v" + `)` + pinVersion + `(?:[/'` + "`" + `"\s-]|$)`)
	default:
		return regexp.MustCompile(`may-journal/fitness-runner@(?:go/)?v` + pinVersion + `(?:[\s'"#]|$)`)
	}
}

func replacePins(path, text, version string) string {
	return referencePattern(path).ReplaceAllStringFunc(text, func(reference string) string {
		reference = pinTag.ReplaceAllString(reference, distribution.ReleaseTag(version))
		return pinNumber.ReplaceAllString(reference, version)
	})
}

type pinEdit struct {
	path string
	text string
}

// UpdatePins advances every supported release reference, rejecting stale runs
// before writing any files. Repeating a completed upgrade makes no changes.
func (c Config) UpdatePins(version string) ([]string, error) {
	if !pinVersionPattern.MatchString(version) {
		return nil, fmt.Errorf("invalid release version %q: expected numeric major.minor.patch", version)
	}
	edits, err := c.preparePinEdits(version)
	if err != nil {
		return nil, err
	}
	return c.writePinEdits(edits)
}

func (c Config) preparePinEdits(version string) ([]pinEdit, error) {
	var edits []pinEdit
	for _, path := range PinFiles() {
		edit, err := c.preparePinEdit(path, version)
		if err != nil {
			return nil, err
		}
		if edit.path != "" {
			edits = append(edits, edit)
		}
	}
	return edits, nil
}

func (c Config) preparePinEdit(path, version string) (pinEdit, error) {
	data, err := os.ReadFile(filepath.Join(c.Root, path))
	if err != nil {
		return pinEdit{}, err
	}
	text := string(data)
	if err := checkPinVersions(path, text, version); err != nil {
		return pinEdit{}, err
	}
	updated := replacePins(path, text, version)
	if updated == text {
		return pinEdit{}, nil
	}
	return pinEdit{path: path, text: updated}, nil
}

func checkPinVersions(path, text, target string) error {
	for _, reference := range referencePattern(path).FindAllString(text, -1) {
		current := pinNumber.FindString(reference)
		if comparePinVersions(current, target) > 0 {
			return fmt.Errorf("%w: refusing to downgrade %s from %s to %s", ErrPinDowngrade, path, current, target)
		}
	}
	return nil
}

// Numeric components are compared by length first to avoid integer overflow.
func comparePinVersions(a, b string) int {
	left, right := strings.Split(a, "."), strings.Split(b, ".")
	for i := range left {
		if len(left[i]) != len(right[i]) {
			return len(left[i]) - len(right[i])
		}
		if order := strings.Compare(left[i], right[i]); order != 0 {
			return order
		}
	}
	return 0
}

func (c Config) writePinEdits(edits []pinEdit) ([]string, error) {
	var changed []string
	for _, edit := range edits {
		if err := os.WriteFile(filepath.Join(c.Root, edit.path), []byte(edit.text), 0600); err != nil {
			return changed, err
		}
		changed = append(changed, edit.path)
	}
	return changed, nil
}
