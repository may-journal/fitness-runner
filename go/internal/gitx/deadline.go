package gitx

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Deadline is how long one git call may run before it is stopped. Ten
// seconds covers `git ls-files` on a monorepo. A call past it is stalled,
// such as one on a network file system or on an index another process holds.
const Deadline = 10 * time.Second

// StalledError is a git call stopped at Deadline. It names the call, so the
// runner can end a run as a setup error that says which git call stalled.
type StalledError struct {
	Args []string
}

func (e *StalledError) Error() string {
	return fmt.Sprintf("git %s did not finish within %s", strings.Join(e.Args, " "), Deadline)
}

// Stalled reports whether err, or an error it wraps, is a StalledError.
func Stalled(err error) bool {
	var stalled *StalledError
	return errors.As(err, &stalled)
}

// TrackedFiles returns the tracked paths, slash-separated and relative to
// the repo, as `git ls-files -z --cached` prints them.
func (r Repo) TrackedFiles() ([]string, error) {
	out, err := r.run("ls-files", "-z", "--cached")
	if err != nil {
		return nil, err
	}
	return nulList(out), nil
}

// GitPath returns the absolute path git uses for name inside the git
// directory, such as index or hooks/commit-msg.
func (r Repo) GitPath(name string) (string, error) {
	out, err := r.run("rev-parse", "--path-format=absolute", "--git-path", name)
	return strings.TrimSpace(out), err
}

// Toplevel returns the root of the work tree holding the repo's path.
func (r Repo) Toplevel() (string, error) {
	out, err := r.run("rev-parse", "--show-toplevel")
	return strings.TrimSpace(out), err
}

// SetConfig sets one key in the repo's local git config.
func (r Repo) SetConfig(key, value string) error {
	_, err := r.run("config", key, value)
	return err
}

// Config returns key's value as git resolves it from the repo's config, ""
// when unset.
func (r Repo) Config(key string) (string, error) {
	out, err := r.run("config", key)
	return strings.TrimSpace(out), err
}

// TracksDir reports whether HEAD tracks dir as a folder.
func (r Repo) TracksDir(dir string) (bool, error) {
	out, err := r.run("ls-tree", "-d", "--name-only", "HEAD", "--", dir)
	return strings.TrimSpace(out) == dir, err
}

// CheckAttr returns `git check-attr -z` output for attr on files: path,
// attribute and value, each NUL-terminated.
func (r Repo) CheckAttr(attr string, files []string) (string, error) {
	return r.input(strings.Join(files, "\x00")+"\x00", "check-attr", "-z", "--stdin", attr)
}

// ToPush returns the commits a push of local sends: those no origin
// branch holds, and, unless remote is empty, not reachable from remote.
func (r Repo) ToPush(local, remote string) ([]string, error) {
	args := []string{"rev-list", local, "--not", "--remotes=origin"}
	if remote != "" {
		args = append(args, remote)
	}
	return r.names(args...)
}

// Log formats exactly the given commits, in order, in one git run.
// --no-walk reads only those commits, not their ancestors, and --stdin takes
// them however many a push holds.
func (r Repo) Log(commits []string, format string) (string, error) {
	return r.input(strings.Join(commits, "\n")+"\n", "log", "--no-walk=unsorted", "--stdin", "--format="+format)
}

// nulList splits git's NUL-terminated `-z` output into its entries,
// dropping empty ones.
func nulList(out string) []string {
	var entries []string
	for _, entry := range strings.Split(out, "\x00") {
		if entry != "" {
			entries = append(entries, entry)
		}
	}
	return entries
}
