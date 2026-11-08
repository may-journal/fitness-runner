package gitx

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// ShowAll returns each of objects, named as rev:path or :path, in one `git
// cat-file --batch` call. An object git lacks, or whose name holds a newline, is left out.
func ShowAll(root string, objects []string) (map[string]string, error) {
	asked := batchNames(objects)
	if len(asked) == 0 {
		return map[string]string{}, nil
	}
	out, err := Repo(root).input(strings.Join(asked, "\n")+"\n", "cat-file", "--batch")
	if err != nil {
		return map[string]string{}, err
	}
	return readObjects(out, asked), nil
}

// Messages returns the full message of each of commits, in their order, read
// with one `git log` call.
func Messages(root string, commits []string) ([]string, error) {
	if len(commits) == 0 {
		return nil, nil
	}
	out, err := Repo(root).input(strings.Join(commits, "\n")+"\n", "log", "--no-walk=unsorted", "--stdin", "-z", "--format=%B")
	if err != nil {
		return nil, err
	}
	messages := splitMessages(out)
	if len(messages) != len(commits) {
		return nil, checkkit.Internal(fmt.Errorf("git log read %d messages for %d commits", len(messages), len(commits)))
	}
	return messages, nil
}

// RecentMessages returns the messages of the newest n commits from HEAD,
// HEAD's first, read with one `git log` call; none before the first commit.
func RecentMessages(root string, n int) []string {
	out, err := Repo(root).run("log", "-n", strconv.Itoa(n), "-z", "--format=%B")
	if err != nil {
		return nil
	}
	return splitMessages(out)
}

// splitMessages splits `git log -z` output into its messages.
func splitMessages(out string) []string {
	if out == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
}

// HasAll reports which of objects git holds, in one `git cat-file
// --batch-check` call that reads no file.
func HasAll(root string, objects []string) (map[string]bool, error) {
	held := map[string]bool{}
	asked := batchNames(objects)
	if len(asked) == 0 {
		return held, nil
	}
	out, err := Repo(root).input(strings.Join(asked, "\n")+"\n", "cat-file", "--batch-check")
	answers := strings.Split(out, "\n")
	for i, name := range asked {
		if err == nil && i < len(answers) {
			_, held[name] = objectSize(answers[i])
		}
	}
	return held, err
}

// batchNames keeps each object name once, dropping any a batch line cannot
// carry.
func batchNames(objects []string) []string {
	var asked []string
	seen := map[string]bool{}
	for _, name := range objects {
		if !seen[name] && !strings.Contains(name, "\n") {
			seen[name] = true
			asked = append(asked, name)
		}
	}
	return asked
}

// readObjects pairs each of asked with its answer in out, in order, keeping
// the objects git held.
func readObjects(out string, asked []string) map[string]string {
	found := map[string]string{}
	for _, name := range asked {
		body, rest, ok := nextObject(out)
		if ok {
			found[name] = body
		}
		out = rest
	}
	return found
}

// nextObject reads git's answer for one object from the start of out: its
// text and whether git held it, then the answers left.
func nextObject(out string) (body, rest string, ok bool) {
	header, rest, _ := strings.Cut(out, "\n")
	size, ok := objectSize(header)
	if !ok || size > len(rest) {
		return "", rest, false
	}
	return rest[:size], strings.TrimPrefix(rest[size:], "\n"), true
}

// objectSize reads a batch header, as `<id> blob 12`, for the object's size.
// A missing object's header, as `<name> missing`, ends in no size.
func objectSize(header string) (int, bool) {
	fields := append(strings.Fields(header), "", "", "")
	size, err := strconv.Atoi(fields[2])
	return size, err == nil && fields[3] == ""
}
