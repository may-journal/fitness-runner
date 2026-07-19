// Command fitness-check-markdown-links requires every relative link in
// every markdown file to resolve to an existing file or directory. Inline
// links, images, and reference definitions are checked; absolute URLs
// (any scheme), protocol-relative URLs, and fragment-only links are never
// touched, so the check is fully offline and deterministic. A `#fragment`
// suffix is stripped before resolution; anchors themselves are not
// validated. Links inside fenced code blocks and inline code spans are
// ignored.
package main

import (
	"fmt"
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

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "markdown-links"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	errs, count, err := walkfs.ScanFiles(root, []string{".md"}, func(rel, content string) []string {
		return scan(root, rel, content)
	})
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errs) > 0 {
		return checkkit.Fail(count, errs...), nil
	}
	return checkkit.Pass(count), nil
}

// scan walks one file's lines, skipping fenced code, and validates every
// link target found outside code spans.
func scan(root, rel, content string) []string {
	var errs []string
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
func judgeLine(root, rel, line string, lineNo int) []string {
	var errs []string
	for _, target := range targets(codeSpanRe.ReplaceAllString(line, "")) {
		if msg, ok := validate(root, rel, target); !ok {
			errs = append(errs, fmt.Sprintf("%s:%d: %s", rel, lineNo, msg))
		}
	}
	return errs
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
	return target, target != ""
}
