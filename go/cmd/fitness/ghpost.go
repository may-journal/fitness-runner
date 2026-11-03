package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// ghPost is a gh call that posts a body: what it posts and the body.
type ghPost struct {
	// kind names what posts, such as "pr" or "issue comment".
	kind string
	// comment marks a comment or review, which meets the comment checks.
	comment bool
	body    string
}

// ghPostKinds maps each gh call that posts a body to whether that body is a
// comment; the rest are issue or PR descriptions.
var ghPostKinds = map[string]bool{
	"issue create": false, "issue edit": false, "pr create": false, "pr edit": false,
	"issue comment": true, "pr comment": true, "pr review": true,
}

// ghInlineFlags give a body inline, and ghFileFlags name a file holding it.
var (
	ghInlineFlags = []string{"--body", "-b"}
	ghFileFlags   = []string{"--body-file", "-F"}
)

// ghCall holds what reading one gh call's body needs. That is its folder,
// its stdin, and the calls before it on the command line.
type ghCall struct {
	cwd     string
	stdin   *shellWord
	earlier []shellCall
}

// ghPostIn finds the first gh issue or PR call in a shell command whose body
// it can read in full. A body built from a variable or another command is
// unknown until the shell runs, so that call is not judged.
func ghPostIn(command, cwd string) (ghPost, bool) {
	if !strings.Contains(command, "gh") {
		return ghPost{}, false
	}
	calls := shellCalls(command)
	for i, call := range calls {
		cwd = afterCd(call, cwd)
		if post, ok := ghPostOf(call.words, ghCall{cwd: cwd, stdin: call.stdin, earlier: calls[:i]}); ok {
			return post, true
		}
	}
	return ghPost{}, false
}

// afterCd is the folder the next call runs in after call, which may cd.
func afterCd(call shellCall, cwd string) string {
	words := call.words
	if len(words) != 2 || words[0].text != "cd" || !words[1].known {
		return cwd
	}
	if filepath.IsAbs(words[1].text) {
		return words[1].text
	}
	return filepath.Join(cwd, words[1].text)
}

// ghPostOf reads the post one shell call's words make, when it is a gh call
// with a known body.
func ghPostOf(words []shellWord, call ghCall) (ghPost, bool) {
	words = leadingCommand(words)
	if len(words) < 3 || filepath.Base(words[0].text) != "gh" {
		return ghPost{}, false
	}
	words = skipRepoFlag(words[1:])
	if len(words) < 2 {
		return ghPost{}, false
	}
	kind := words[0].text + " " + words[1].text
	comment, posts := ghPostKinds[kind]
	body, ok := ghBody(words[2:], call)
	kind = strings.TrimSuffix(strings.TrimSuffix(kind, " create"), " edit")
	return ghPost{kind: kind, comment: comment, body: body}, posts && ok
}

// leadingCommand drops the variable assignments and the command or env
// word that may lead a call.
func leadingCommand(words []shellWord) []shellWord {
	for len(words) > 0 && words[0].known && (assignment.MatchString(words[0].text) || slices.Contains(commandPrefixes, words[0].text)) {
		words = words[1:]
	}
	return words
}

// commandPrefixes run the command that follows them.
var commandPrefixes = []string{"command", "env"}

// skipRepoFlag drops a -R or --repo flag, and its value, that comes before
// the gh subcommand.
func skipRepoFlag(args []shellWord) []shellWord {
	width := 0
	if len(args) > 0 {
		width = repoFlagWidth(args[0].text)
	}
	return args[min(width, len(args)):]
}

// repoFlagWidth is how many words a leading repo flag takes: two for
// -R owner/repo, one for --repo=owner/repo, else none.
func repoFlagWidth(word string) int {
	if word == "-R" || word == "--repo" {
		return 2
	}
	if strings.HasPrefix(word, "--repo=") {
		return 1
	}
	return 0
}

// ghBody reads the body a gh call's flags give: inline, from a file, or from
// the call's stdin when the file is -.
func ghBody(args []shellWord, call ghCall) (string, bool) {
	for i, arg := range args {
		if name, value, ok := flagValue(args, i); arg.known && ok {
			return flagBody(name, value, call)
		}
	}
	return "", false
}

// flagBody reads the body one flag gives, when its value is known.
func flagBody(name string, value shellWord, call ghCall) (string, bool) {
	if !value.known {
		return "", false
	}
	if slices.Contains(ghInlineFlags, name) {
		return value.text, true
	}
	return call.bodyFile(value.text)
}

// flagValue reads args[i] as a body flag, given as --flag=value or as
// --flag followed by its value.
func flagValue(args []shellWord, i int) (name string, value shellWord, ok bool) {
	name, inline, hasValue := strings.Cut(args[i].text, "=")
	if !slices.Contains(ghInlineFlags, name) && !slices.Contains(ghFileFlags, name) {
		return "", shellWord{}, false
	}
	if hasValue {
		return name, shellWord{text: inline, known: true}, true
	}
	if i+1 < len(args) {
		return name, args[i+1], true
	}
	return name, shellWord{}, true
}

// bodyFile reads a --body-file value: the call's stdin for -, else the file.
// A file an earlier call on the line names may be written before gh reads
// it, so its text is unknown.
func (call ghCall) bodyFile(path string) (string, bool) {
	if path == "-" {
		return stdinText(call.stdin)
	}
	if mentions(call.earlier, path) {
		return "", false
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(call.cwd, path)
	}
	data, err := os.ReadFile(path)
	return string(data), err == nil
}

// mentions reports whether any call names path as a word or a redirection
// target.
func mentions(calls []shellCall, path string) bool {
	for _, call := range calls {
		for _, word := range call.words {
			if strings.TrimLeft(word.text, "0123456789>&") == path {
				return true
			}
		}
	}
	return false
}

// stdinText is the text a heredoc or here-string feeds a call, when known.
func stdinText(stdin *shellWord) (string, bool) {
	if stdin == nil {
		return "", false
	}
	return stdin.text, stdin.known
}

// assignment matches a shell variable assignment word, such as GH_REPO=x.
var assignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
