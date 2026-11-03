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

// ghPostIn finds the first gh issue or PR call in a shell command whose body
// it can read in full. A body built from a variable or another command is
// unknown until the shell runs, so that call is not judged.
func ghPostIn(command, cwd string) (ghPost, bool) {
	if !strings.Contains(command, "gh") {
		return ghPost{}, false
	}
	for _, call := range shellCalls(command) {
		if post, ok := ghPostOf(call, cwd); ok {
			return post, true
		}
	}
	return ghPost{}, false
}

// ghPostOf reads the post one shell call makes, when it is a gh call with a
// known body.
func ghPostOf(call shellCall, cwd string) (ghPost, bool) {
	words := withoutAssignments(call.words)
	if len(words) < 3 || words[0].text != "gh" {
		return ghPost{}, false
	}
	kind := words[1].text + " " + words[2].text
	comment, posts := ghPostKinds[kind]
	if !posts {
		return ghPost{}, false
	}
	body, ok := ghBody(words[3:], call.stdin, cwd)
	kind = strings.TrimSuffix(strings.TrimSuffix(kind, " create"), " edit")
	return ghPost{kind: kind, comment: comment, body: body}, ok
}

// withoutAssignments drops the variable assignments that lead a call.
func withoutAssignments(words []shellWord) []shellWord {
	for len(words) > 0 && words[0].known && assignment.MatchString(words[0].text) {
		words = words[1:]
	}
	return words
}

// ghBody reads the body a gh call's flags give: inline, from a file, or from
// the call's stdin when the file is -.
func ghBody(args []shellWord, stdin *shellWord, cwd string) (string, bool) {
	for i, arg := range args {
		if name, value, ok := flagValue(args, i); arg.known && ok {
			return flagBody(name, value, stdin, cwd)
		}
	}
	return "", false
}

// flagBody reads the body one flag gives, when its value is known.
func flagBody(name string, value shellWord, stdin *shellWord, cwd string) (string, bool) {
	if !value.known {
		return "", false
	}
	if slices.Contains(ghInlineFlags, name) {
		return value.text, true
	}
	return bodyFile(value.text, stdin, cwd)
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
func bodyFile(path string, stdin *shellWord, cwd string) (string, bool) {
	if path == "-" {
		return stdinText(stdin)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	data, err := os.ReadFile(path)
	return string(data), err == nil
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
