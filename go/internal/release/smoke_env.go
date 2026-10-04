package release

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func prepareSmoke(work string) (smoke, error) {
	test := smoke{work: work, tools: filepath.Join(work, "tools"), cache: filepath.Join(work, "cache"), repo: filepath.Join(work, "repo"), installer: filepath.Join(work, installerName)}
	if cache := os.Getenv("FITNESS_CACHE_DIR"); cache != "" {
		test.cache = cache
	}
	if tools := os.Getenv("FITNESS_TEST_TOOL_PATH"); tools != "" {
		test.tools = tools
	}
	if err := test.prepareDirectories(); err != nil {
		return test, err
	}
	if err := test.linkTools(); err != nil {
		return test, err
	}
	test.env = append(cleanEnvironment(), "PATH="+test.tools, "FITNESS_CACHE_DIR="+test.cache)
	return test, nil
}

func (s smoke) prepareDirectories() error {
	for _, path := range []string{s.tools, s.cache, s.repo} {
		if err := os.MkdirAll(path, 0700); err != nil {
			return err
		}
	}
	return nil
}

func (s smoke) linkTools() error {
	for _, tool := range []string{"bash", "curl", "grep", "shasum", "chmod", "git"} {
		if err := linkTool(s.tools, tool); err != nil {
			return err
		}
	}
	_, err := os.Stat(filepath.Join(s.tools, "go"))
	if !os.IsNotExist(err) {
		return fmt.Errorf("Go must not be on the consumer PATH")
	}
	return nil
}

func linkTool(directory, name string) error {
	target, err := exec.LookPath(name)
	if err != nil {
		return err
	}
	path := filepath.Join(directory, name)
	if _, err := os.Lstat(path); err == nil {
		return nil
	}
	return os.Symlink(target, path)
}

func cleanEnvironment() []string {
	var env []string
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GIT_") && !strings.HasPrefix(entry, "GITHUB_ACTIONS=") {
			env = append(env, entry)
		}
	}
	// Invalid fixtures must fail without publishing failures for a passing smoke test.
	// The parent verifier keeps its environment and reports the final outcome.
	return append(env, "GITHUB_ACTIONS=false")
}

func verifyHash(expected string, data []byte) error {
	actual := fmt.Sprintf("%x", sha256.Sum256(data))
	if actual != expected {
		return fmt.Errorf("installer checksum mismatch")
	}
	return nil
}
