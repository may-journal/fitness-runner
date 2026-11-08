// cspell:ignore nohup caffeinate
package main

import (
	"cmp"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/metadata"
)

// ghPost is one call that posts text. It holds the checks its bodies get,
// its folder, the repo it names, and each body.
type ghPost struct {
	checks []string
	dir    string
	// repo is the owner/repo the call names, "" when it names none, or
	// unknownRepo when the shell builds the name.
	repo  string
	parts []postPart
}

// unknownRepo stands for a repo name the shell builds, which no checkout
// matches.
const unknownRepo = "?"

// postPart is one body a post holds, such as a review's summary or one of
// its line comments, with a name for it. unread marks a body the hook cannot
// read before the shell runs.
type postPart struct {
	subject, text string
	unread        bool
	// root and args are where and how the hook judges the body, with root
	// "" for the org defaults.
	root string
	args []string
}

// ghKind is a gh command that posts a body. It names the checks that body
// gets, and which flags give it inline or name its file.
type ghKind struct {
	checks        func(words []shellWord) []string
	inline, files []string
}

// The flag sets gh commands give a body with, and the checks each body gets
// on GitHub.
var (
	bodyInline, bodyFiles = []string{"--body", "-b"}, []string{"--body-file", "-F"}
	prFlags               = ghKind{checks: always(metadata.PRBodyChecks), inline: bodyInline, files: bodyFiles}
	issueFlags            = ghKind{checks: issueChecks, inline: bodyInline, files: bodyFiles}
	commentFlags          = ghKind{checks: always(metadata.CommentChecks), inline: bodyInline, files: bodyFiles}
	closeFlags            = ghKind{checks: always(metadata.CommentChecks), inline: []string{"--comment", "-c"}}
)

// ghPostKinds are the gh issue and PR commands that post a body.
var ghPostKinds = map[string]ghKind{
	"issue create": issueFlags, "issue edit": issueFlags, "pr create": prFlags, "pr edit": prFlags,
	"issue comment": commentFlags, "pr comment": commentFlags, "pr review": commentFlags,
	"issue close": closeFlags, "pr close": closeFlags,
}

// always is a checks function that names the same checks for every call.
func always(checks []string) func([]shellWord) []string {
	return func([]shellWord) []string { return checks }
}

// issueChecks are the checks issue-check gives the body, by the kind a
// --label or a [Kind] title prefix names.
func issueChecks(words []shellWord) []string {
	return kindChecks(issueKinds(words))
}

// kindChecks are the checks of the first kind that names some, or the doc
// checks alone when none does.
func kindChecks(kinds []string) []string {
	for _, kind := range kinds {
		if checks := metadata.IssueBodyChecks(kind); checks != nil {
			return checks
		}
	}
	return metadata.IssueBodyChecks("")
}

// issueKinds are the labels a call's label flags give, and the kind its
// title prefix names.
func issueKinds(words []shellWord) []string {
	var kinds []string
	for i := range words {
		if _, value, ok := flagValue(words, i, []string{"--label", "-l", "--add-label"}); ok {
			kinds = append(kinds, strings.Split(value.text, ",")...)
		}
		if _, value, ok := flagValue(words, i, []string{"--title", "-t"}); ok {
			kinds = append(kinds, titleKind.FindString(value.text))
		}
	}
	return cleanKinds(kinds)
}

// cleanKinds trims each kind of its spaces and brackets, dropping empty ones.
func cleanKinds(kinds []string) []string {
	var clean []string
	for _, kind := range kinds {
		if kind = strings.Trim(strings.TrimSpace(kind), "[]"); kind != "" {
			clean = append(clean, kind)
		}
	}
	return clean
}

// titleKind matches the [Kind] prefix the issue templates give a title.
var titleKind = regexp.MustCompile(`^\[(\w+)\]`)

// ghCall holds what reading one gh call's body needs. That is its folder,
// its stdin, and the calls before it on the command line.
type ghCall struct {
	cwd     string
	stdin   *shellWord
	earlier []shellCall
	// repo is the GH_REPO an assignment before the call names.
	repo string
}

// ghPostsIn finds every issue, PR or api call in a shell command that posts
// a body. That includes the calls in a loop, a wrapper or a `bash -c` script.
func ghPostsIn(command, cwd string) []ghPost {
	return postsIn(command, cwd, "")
}

// postsIn is ghPostsIn for a script that runs with the GH_REPO repo names.
func postsIn(command, cwd, inherited string) []ghPost {
	if !strings.Contains(command, "gh") {
		return nil
	}
	var posts []ghPost
	calls := shellCalls(command)
	for i, call := range calls {
		cwd = afterCd(call, cwd)
		words, repo := unwrap(call.words)
		repo = cmp.Or(repo, inherited)
		if script, ok := shellScript(words); ok {
			posts = append(posts, postsIn(script, cwd, repo)...)
			continue
		}
		if post, ok := ghPostOf(words, ghCall{cwd: cwd, stdin: call.stdin, earlier: calls[:i], repo: repo}); ok {
			posts = append(posts, post)
		}
	}
	return posts
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
// that posts a body.
func ghPostOf(words []shellWord, call ghCall) (ghPost, bool) {
	if len(words) < 3 || filepath.Base(words[0].text) != "gh" {
		return ghPost{}, false
	}
	target := words[1:]
	words = skipRepoFlag(target)
	var post ghPost
	var ok bool
	if len(words) > 0 && words[0].text == "api" {
		post, ok = apiPost(words[1:], call)
	} else {
		post, ok = commandPost(words, call)
	}
	post.dir, post.repo = call.cwd, targetRepo(target, call.repo, post.repo)
	return post, ok
}

// commandPost reads the post a gh issue or PR command makes, from the words
// after gh. A command with no body flag posts nothing to judge.
func commandPost(words []shellWord, call ghCall) (ghPost, bool) {
	if len(words) < 2 {
		return ghPost{}, false
	}
	name := words[0].text + " " + words[1].text
	kind, posts := ghPostKinds[name]
	if !posts {
		return ghPost{}, false
	}
	body, given, ok := ghBody(words[2:], call, kind)
	subject := "the " + strings.TrimSuffix(strings.TrimSuffix(name, " create"), " edit") + " body"
	return ghPost{checks: kind.checks(words[2:]), parts: []postPart{{subject: subject, text: body, unread: !ok}}}, given
}

// unwrap drops the assignments, shell keywords and wrappers, such as
// timeout and xargs, that lead the command a call runs. It also returns the
// GH_REPO an assignment names.
func unwrap(words []shellWord) ([]shellWord, string) {
	repo := ""
	for len(words) > 0 {
		word := words[0]
		if value, ok := strings.CutPrefix(word.text, "GH_REPO="); ok {
			repo = knownRepo(value, word.known)
		}
		width := wrapperWidth(words)
		if width == 0 {
			return words, repo
		}
		words = words[min(width, len(words)):]
	}
	return words, repo
}

// wrapperWidth is how many words lead the call before the command it runs,
// or 0 when the first word is that command.
func wrapperWidth(words []shellWord) int {
	word := words[0]
	if assignment.MatchString(word.text) {
		return 1
	}
	if !word.known {
		return 0
	}
	if slices.Contains(commandPrefixes, word.text) {
		return 1
	}
	if valued, ok := wrapperOptions[word.text]; ok {
		return 1 + optionsWidth(words[1:], valued.options) + valued.operands
	}
	return 0
}

// wrapper is a command that runs the command after its options and
// operands. It lists the options that take a value, and counts its operands.
type wrapper struct {
	options  []string
	operands int
}

// wrapperOptions are the wrappers by name; timeout reads a duration first.
// An assignment env sets after its options is dropped as the next word.
var wrapperOptions = map[string]wrapper{
	"timeout":    {options: []string{"-s", "--signal", "-k", "--kill-after"}, operands: 1},
	"xargs":      {options: []string{"-I", "-n", "-P", "-L", "-d", "-E", "-s", "-a"}},
	"env":        {options: []string{"-u", "--unset", "-C", "--chdir", "-S"}},
	"sudo":       {options: []string{"-u", "-g", "-C", "-D", "-h", "-p", "-U", "-r", "-t"}},
	"time":       {options: []string{"-f", "-o"}},
	"nice":       {options: []string{"-n"}},
	"nohup":      {},
	"caffeinate": {options: []string{"-t", "-w"}},
}

// commandPrefixes are the shell keywords and wrappers that run the command
// after them with no value of their own.
var commandPrefixes = []string{"command", "exec", "do", "then", "else", "{", "!"}

// optionsWidth is how many words the leading options take. Each option in
// valued reads the next word as its value.
func optionsWidth(words []shellWord, valued []string) int {
	i := 0
	for i < len(words) && strings.HasPrefix(words[i].text, "-") {
		if slices.Contains(valued, words[i].text) {
			i++
		}
		i++
	}
	return i
}

// shellScript is the script a `bash -c`, `sh -c` or `zsh -c` call runs,
// when the shell knows it before it runs. The script is the first word
// after the options, once a short option cluster holds c.
func shellScript(words []shellWord) (string, bool) {
	if len(words) < 3 || !slices.Contains([]string{"bash", "sh", "zsh"}, filepath.Base(words[0].text)) {
		return "", false
	}
	i := 1 + slices.IndexFunc(words[1:], func(w shellWord) bool { return !strings.HasPrefix(w.text, "-") })
	if i < 2 || !slices.ContainsFunc(words[1:i], commandFlag) {
		return "", false
	}
	return words[i].text, words[i].known
}

// commandFlag reports whether a word is a short option cluster holding c,
// such as -c or -lc.
func commandFlag(w shellWord) bool {
	return shortFlags.MatchString(w.text) && strings.Contains(w.text, "c")
}

// shortFlags matches a word of single-letter shell options, such as -lc.
var shortFlags = regexp.MustCompile(`^-[a-z]+$`)

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

// targetRepo is the repo a gh call posts to. That is its -R flag, a GitHub
// URL it names, the repo an endpoint names, or GH_REPO.
func targetRepo(words []shellWord, assigned, endpoint string) string {
	for i, word := range words {
		if _, value, ok := flagValue(words, i, []string{"-R", "--repo"}); ok {
			return knownRepo(value.text, value.known)
		}
		if m := repoURL.FindStringSubmatch(word.text); m != nil {
			return knownRepo(m[1], word.known)
		}
	}
	return cmp.Or(endpoint, assigned)
}

// knownRepo is the owner/repo a value names, without a host, or unknownRepo
// when the shell builds it.
func knownRepo(value string, known bool) string {
	if !known {
		return unknownRepo
	}
	parts := strings.Split(strings.TrimSuffix(value, "/"), "/")
	if len(parts) < 2 {
		return unknownRepo
	}
	return strings.ToLower(strings.Join(parts[len(parts)-2:], "/"))
}

// repoURL matches a word that is a GitHub issue or PR URL, capturing its
// owner/repo. A URL inside a body names no target.
var repoURL = regexp.MustCompile(`^https://github\.com/([^/\s]+/[^/\s]+)/(?:pull|issues)/\d+/?$`)

// ghBody reads the body a gh call's flags give: inline, from a file, or from
// the call's stdin when the file is -. given is false when no flag gives a
// body, and ok false when the shell builds it.
func ghBody(args []shellWord, call ghCall, kind ghKind) (body string, given, ok bool) {
	flags := append(slices.Clone(kind.inline), kind.files...)
	for i := range args {
		if name, value, isFlag := flagValue(args, i, flags); isFlag {
			body, ok = flagBody(slices.Contains(kind.inline, name), value, call)
			return body, true, ok
		}
	}
	return "", false, false
}

// flagBody reads the body one flag gives, when known. That is the value for
// an inline flag, else the file it names.
func flagBody(inline bool, value shellWord, call ghCall) (string, bool) {
	if !value.known {
		return "", false
	}
	if inline {
		return value.text, true
	}
	return call.bodyFile(value.text)
}

// flagValue reads args[i] as one of flags, given as --flag=value or as
// --flag followed by its value.
func flagValue(args []shellWord, i int, flags []string) (name string, value shellWord, ok bool) {
	name, inline, hasValue := strings.Cut(args[i].text, "=")
	if !slices.Contains(flags, name) {
		return "", shellWord{}, false
	}
	if hasValue {
		return name, shellWord{text: inline, known: args[i].known}, true
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

// stdinText is the text a heredoc or here-string feeds a call, when known. A
// pipe feeds text the hook cannot read.
func stdinText(stdin *shellWord) (string, bool) {
	if stdin == nil {
		return "", false
	}
	return stdin.text, stdin.known
}

// assignment matches a shell variable assignment word, such as GH_REPO=x.
var assignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
