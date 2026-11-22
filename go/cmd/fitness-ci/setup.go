// cspell:ignore sdkmanrc sdkman

package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/pkg/actions"
)

func init() {
	register("find-go-module", "Write the shallowest go.mod, or nothing, as file, so setup-go can use the runner's Go.", []string{"file"}, findGoModule)
	register("find-jdk-version-file", "Write the JDK version file as file, where a .sdkmanrc or .tool-versions counts only with a java line.", []string{"file"}, findJDKVersionFile)
	register("install-swiftlint", "Install SwiftLint VERSION's ASSET, checking the zip against its published SHA-256 and caching it under a versioned name.", nil, installSwiftLint)
	register("check-command-input", "Fail unless FITNESS_COMMAND is issue-check or plan-check, the two commands the issue-check workflow runs.", nil, checkCommandInput)
}

func findGoModule(step actions.Step, _ []string) error {
	listed, err := actions.Git("ls-files", "--", "go.mod", "*/go.mod")
	if err != nil {
		return err
	}
	// One path per line, since a path may hold a space.
	files := strings.FieldsFunc(listed, func(r rune) bool { return r == '\n' })
	slices.SortFunc(files, byDepth)
	return step.Output("file", strings.Join(files[:min(len(files), 1)], ""))
}

// byDepth orders paths shallowest first, then by name.
func byDepth(a, b string) int {
	if d := strings.Count(a, "/") - strings.Count(b, "/"); d != 0 {
		return d
	}
	return strings.Compare(a, b)
}

var (
	// sdkmanJava is a .sdkmanrc line that pins Java.
	sdkmanJava = regexp.MustCompile(`^java[[:space:]]*=`)
	// toolVersionsJava is a .tool-versions line that pins Java.
	toolVersionsJava = regexp.MustCompile(`^java[[:space:]]`)
)

func findJDKVersionFile(step actions.Step, _ []string) error {
	file := ""
	switch {
	case exists(".java-version"):
		file = ".java-version"
	case hasLineMatching(".sdkmanrc", sdkmanJava):
		file = ".sdkmanrc"
	case hasLineMatching(".tool-versions", toolVersionsJava):
		file = ".tool-versions"
	}
	return step.Output("file", file)
}

// exists reports whether path names a regular file.
func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// hasLineMatching reports whether a line of the file at path matches re; a
// missing file has none.
func hasLineMatching(path string, re *regexp.Regexp) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if re.MatchString(scanner.Text()) {
			return true
		}
	}
	return false
}

func checkCommandInput(actions.Step, []string) error {
	switch command := env("FITNESS_COMMAND"); command {
	case "issue-check", "plan-check":
		return nil
	default:
		return actions.Fail("command must be issue-check or plan-check, not " + command)
	}
}

func installSwiftLint(actions.Step, []string) error {
	zip, err := swiftLintZip(env("VERSION"), env("ASSET"))
	if err != nil {
		return err
	}
	dir := filepath.Join(env("RUNNER_TEMP"), "swiftlint")
	if err := unpackSwiftLint(zip, dir); err != nil {
		return err
	}
	if err := actions.Path(dir); err != nil {
		return err
	}
	return actions.Run(filepath.Join(dir, "swiftlint"), "version")
}

// swiftLintZip is the cached zip of asset in release version, downloaded
// first unless the cached one matches its published SHA-256.
func swiftLintZip(version, asset string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	zip := filepath.Join(home, ".cache", "fitness-swiftlint", asset+"-"+version+".zip")
	want, err := swiftLintDigest(version, asset)
	if err != nil {
		return "", err
	}
	if sum, _ := fileSHA256(zip); sum != want {
		return zip, downloadSwiftLint(version, asset, zip, want)
	}
	return zip, nil
}

// unpackSwiftLint unzips zip into a fresh dir and makes swiftlint runnable.
func unpackSwiftLint(zip, dir string) error {
	return firstError(
		func() error { return os.RemoveAll(dir) },
		func() error { return actions.Run("unzip", "-q", "-o", zip, "-d", dir) },
		func() error { return actions.MakeExecutable(filepath.Join(dir, "swiftlint")) },
	)
}

// swiftLintDigest is the SHA-256 SwiftLint publishes for asset's zip in
// release version, asked for up to three times.
func swiftLintDigest(version, asset string) (string, error) {
	query := fmt.Sprintf(`.assets[] | select(.name == "%s.zip") | .digest`, asset)
	var want string
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		if want, err = actions.Gh("api", "repos/realm/SwiftLint/releases/tags/"+version, "--jq", query); err == nil {
			break
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	want = strings.TrimPrefix(want, "sha256:")
	if want == "" {
		return "", actions.Fail(fmt.Sprintf("SwiftLint %s publishes no SHA-256 for %s.zip", version, asset))
	}
	return want, nil
}

// downloadSwiftLint downloads asset's zip to zip through a part file only
// this job writes, keeping it only when it matches want.
func downloadSwiftLint(version, asset, zip, want string) error {
	if err := os.MkdirAll(filepath.Dir(zip), 0o755); err != nil {
		return err
	}
	part := fmt.Sprintf("%s.%s.%s.%d", zip, env("GITHUB_RUN_ID"), env("GITHUB_JOB"), os.Getpid())
	url := fmt.Sprintf("https://github.com/realm/SwiftLint/releases/download/%s/%s.zip", version, asset)
	if err := actions.Run("curl", "-fsSL", "--retry", "3", "-o", part, url); err != nil {
		return err
	}
	if sum, err := fileSHA256(part); err != nil || sum != want {
		_ = os.Remove(part)
		return actions.Fail(fmt.Sprintf("SwiftLint %s %s.zip does not match its published SHA-256", version, asset))
	}
	return os.Rename(part, zip)
}

// fileSHA256 is the hex SHA-256 of the file at path.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	_, err = io.Copy(hash, f)
	return hex.EncodeToString(hash.Sum(nil)), err
}
