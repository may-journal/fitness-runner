package distribution

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

func ActionOptions(options Options) (Options, error) {
	if !options.GitHubAction {
		return options, nil
	}
	return actionInputs(options, os.Getenv)
}

func actionInputs(options Options, getenv func(string) string) (Options, error) {
	if version := getenv("FITNESS_VERSION"); version != "" {
		options.Version = version
	} else if err := inMajor(options.Version, getenv("FITNESS_ACTION_REF")); err != nil {
		return options, err
	}
	args, err := actionSelection(getenv)
	if err != nil {
		return options, err
	}
	if len(args) > 0 {
		options.Args = args
	}
	return withModes(options, getenv)
}

// majorRef is an action ref that names a major version, such as v1.
var majorRef = regexp.MustCompile(`^v[0-9]+$`)

// inMajor fails when the action runs at a major ref, such as v1, and the
// installer's newest release is outside it.
func inMajor(version, ref string) error {
	if majorRef.MatchString(ref) && !strings.HasPrefix(version, ref[1:]+".") {
		return fmt.Errorf("the newest Fitness release %s is outside %s", version, ref)
	}
	return nil
}

// withModes reads the install-only and with-checks inputs into options.
func withModes(options Options, getenv func(string) string) (Options, error) {
	var err error
	if options.InstallOnly, err = boolInput(getenv("FITNESS_INSTALL_ONLY")); err != nil {
		return options, err
	}
	options.WithChecks, options.PinnedOnly, err = checksMode(getenv("FITNESS_WITH_CHECKS"))
	return options, err
}

// boolInput reads a true or false action input, false when empty.
func boolInput(value string) (bool, error) {
	if value == "" {
		return false, nil
	}
	return strconv.ParseBool(value)
}

// WriteActionFiles puts the engine on PATH and names it in the bin output.
// With checks installed, later steps get FITNESS_CHECKS_DIR and the
// checks-dir output names that folder.
func WriteActionFiles(bin, checks string) error {
	if strings.ContainsAny(bin+checks, "\r\n") {
		return fmt.Errorf("cache path cannot contain a newline")
	}
	if err := appendActionFile(os.Getenv("GITHUB_PATH"), bin+"\n"); err != nil {
		return err
	}
	if err := appendActionFile(os.Getenv("GITHUB_OUTPUT"), "bin="+bin+"\n"); err != nil || checks == "" {
		return err
	}
	return writeChecksDir(checks)
}

// writeChecksDir names the installed checks folder in the checks-dir output
// and in FITNESS_CHECKS_DIR for later steps.
func writeChecksDir(checks string) error {
	if err := appendActionFile(os.Getenv("GITHUB_OUTPUT"), "checks-dir="+checks+"\n"); err != nil {
		return err
	}
	return appendActionFile(os.Getenv("GITHUB_ENV"), checkkit.ChecksDirEnv+"="+checks+"\n")
}

func appendActionFile(path, value string) error {
	if path == "" {
		return fmt.Errorf("GitHub action output file is missing")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(value)
	return err
}

func actionSelection(getenv func(string) string) ([]string, error) {
	check, checks := getenv("FITNESS_CHECK"), getenv("FITNESS_CHECKS")
	if check != "" && checks != "" {
		return nil, fmt.Errorf("choose action check or checks, not both")
	}
	args := append(withValue("--policy=", getenv("FITNESS_POLICY")), withValue("--checks=", checks)...)
	return append(args, withValue("", check)...), nil
}

// withValue is the single argument flag+value, or nothing when value is empty.
func withValue(flag, value string) []string {
	if value == "" {
		return nil
	}
	return []string{flag + value}
}
