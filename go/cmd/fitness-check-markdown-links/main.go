package check

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

var (
	inlineLinkRe = regexp.MustCompile(`!?\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	refDefRe     = regexp.MustCompile(`^\s*\[[^\]]+\]:\s+(\S+)`)
	codeSpanRe   = regexp.MustCompile("`[^`]+`")
	// skipRe matches targets outside this check's scope: scheme URLs
	// (https:, mailto:, etc.), protocol-relative URLs, and pure fragments.
	skipRe = regexp.MustCompile(`^(?:[a-zA-Z][a-zA-Z0-9+.-]*:|//|#)`)
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "markdown-links", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	return walkfs.ScanFindings(root, []string{".md"}, errorLine, func(rel, content string) []checkkit.Finding {
		return scan(root, rel, content)
	}), nil
}

// scan walks one file's lines, skipping fenced code, and validates every
// link target found outside code spans.
func scan(root, rel, content string) []checkkit.Finding {
	var errs []checkkit.Finding
	inFence := false
	for i, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		errs = append(errs, judgeLine(root, rel, line, i+1)...)
	}
	return errs
}

// judgeLine validates every link target on one line.
func judgeLine(root, rel, line string, lineNo int) []checkkit.Finding {
	var errs []checkkit.Finding
	for _, target := range targets(codeSpanRe.ReplaceAllString(line, "")) {
		if msg, ok := validate(root, rel, target); !ok {
			errs = append(errs, checkkit.Finding{File: rel, Line: lineNo, Rule: rule, Message: msg, Fix: fix})
		}
	}
	return errs
}

// rule names what the check enforces: a relative link resolves.
const rule = "broken-relative-link"

// fix says how to mend a broken relative link.
const fix = "point the link at a file or folder that exists, relative to this file, or remove it"

// errorLine renders a finding as "<path>:<line>: <message>".
func errorLine(f checkkit.Finding) string {
	return fmt.Sprintf("%s:%d: %s", f.File, f.Line, f.Message)
}

// targets extracts link targets from a code-span-masked line: inline links
// and images plus reference-style definitions.
func targets(line string) []string {
	var out []string
	for _, m := range inlineLinkRe.FindAllStringSubmatch(line, -1) {
		out = append(out, m[1])
	}
	if m := refDefRe.FindStringSubmatch(line); m != nil {
		out = append(out, m[1])
	}
	return out
}

// validate resolves one target against the linking file's directory; ok is
// false when a relative target does not exist.
func validate(root, rel, target string) (string, bool) {
	path, ok := resolvable(target)
	if !ok {
		return "", true
	}
	abs := filepath.Join(root, filepath.Dir(filepath.FromSlash(rel)), filepath.FromSlash(path))
	if _, err := os.Stat(abs); err != nil {
		return fmt.Sprintf("broken relative link: %s", target), false
	}
	return "", true
}

// resolvable strips the fragment and reports whether the target is a
// relative path this check validates.
func resolvable(target string) (string, bool) {
	if skipRe.MatchString(target) {
		return "", false
	}
	if i := strings.IndexByte(target, '#'); i >= 0 {
		target = target[:i]
	}
	// A link may percent-encode its path (a space as %20).
	// Check the file it names, falling back to the raw text when it does not decode.
	if decoded, err := url.PathUnescape(target); err == nil {
		target = decoded
	}
	return target, target != ""
}
