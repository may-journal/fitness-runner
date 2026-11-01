package check

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang/toolchain"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
)

// gitLine is git's one line of output in root, or empty when git fails.
func gitLine(root string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// failedAtBase lists the files this check's errors name at base commit rev,
// or none when rev cannot be judged.
func failedAtBase(root, rev string) map[string]bool {
	failed := map[string]bool{}
	if rev == "" {
		return failed
	}
	dir, err := os.MkdirTemp("", "fitness-requirements-base-")
	if err != nil {
		return failed
	}
	defer os.RemoveAll(dir)
	for _, e := range baseErrors(root, rev, dir) {
		if file, _, ok := strings.Cut(e, ":"); ok {
			failed[file] = true
		}
	}
	return failed
}

// baseErrors checks out rev into dir and returns an unscoped run's errors.
func baseErrors(root, rev, dir string) []string {
	self, err := checkkit.Self()
	if err != nil || checkout(root, rev, dir) != nil {
		return nil
	}
	cmd := self.Command("--root", dir)
	cmd.Env = append(toolchain.CleanEnv(os.Environ()), "GITHUB_BASE_REF=")
	out, _ := cmd.Output()
	var res checkkit.Result
	if json.Unmarshal(out, &res) != nil {
		return nil
	}
	return res.Errors
}

// checkout writes only the files this check reads from rev into a repo in dir
// that borrows root's objects, leaving root untouched.
func checkout(root, rev, dir string) error {
	if err := gitIn(dir, nil, "init", "-q"); err != nil {
		return err
	}
	objects := gitLine(root, "rev-parse", "--path-format=absolute", "--git-path", "objects")
	if err := os.WriteFile(filepath.Join(dir, ".git", "objects", "info", "alternates"), []byte(objects+"\n"), 0o600); err != nil {
		return err
	}
	entries, err := judgedEntries(root, rev)
	if err == nil {
		err = gitIn(dir, entries, "update-index", "-z", "--index-info")
	}
	if err != nil {
		return err
	}
	return gitIn(dir, nil, "checkout-index", "-a")
}

// judgedEntries lists rev's requirement docs, test files and .gitattributes
// files as `git ls-tree -r -z` prints them.
func judgedEntries(root, rev string) ([]byte, error) {
	out, err := exec.Command("git", "-C", root, "ls-tree", "-r", "-z", "--full-tree", rev).Output()
	if err != nil {
		return nil, err
	}
	var kept bytes.Buffer
	for _, entry := range bytes.Split(out, []byte{0}) {
		if _, file, ok := strings.Cut(string(entry), "\t"); ok && judged(file) {
			kept.Write(entry)
			kept.WriteByte(0)
		}
	}
	return kept.Bytes(), nil
}

// judged reports whether this check reads file.
func judged(file string) bool {
	return strings.HasPrefix(file, docsDir) || path.Base(file) == ".gitattributes" || lang.TestReads(langs.All, file)
}

// gitIn runs git in dir with stdin as its input, without the hook
// variables that point at the repository being committed.
func gitIn(dir string, stdin []byte, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = toolchain.CleanEnv(os.Environ())
	cmd.Stdin = bytes.NewReader(stdin)
	return cmd.Run()
}
