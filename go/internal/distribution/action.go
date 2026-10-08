package distribution

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	}
	args, err := actionSelection(getenv)
	if err != nil {
		return options, err
	}
	if len(args) > 0 {
		options.Args = args
	}
	value := getenv("FITNESS_INSTALL_ONLY")
	if value == "" {
		value = "false"
	}
	installOnly, err := strconv.ParseBool(value)
	options.InstallOnly = installOnly
	return options, err
}

func WriteActionFiles(bin string) error {
	if strings.ContainsAny(bin, "\r\n") {
		return fmt.Errorf("cache path cannot contain a newline")
	}
	if err := appendActionFile(os.Getenv("GITHUB_PATH"), bin+"\n"); err != nil {
		return err
	}
	return appendActionFile(os.Getenv("GITHUB_OUTPUT"), "bin="+bin+"\n")
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
