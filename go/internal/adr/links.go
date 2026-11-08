package adr

import (
	"path"
	"regexp"
	"strings"
)

// inlineLink matches a Markdown inline link's target; referenceLink matches
// a link reference definition's.
var (
	inlineLink    = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	referenceLink = regexp.MustCompile(`(?m)^(\s*\[[^\]]+\]:\s*)(\S+)`)
)

// Resolver maps the repo path a link points at, and its anchor, to the URL
// that replaces the link. It reports false to keep the link.
type Resolver func(target, anchor string) (string, bool)

// RewriteLinks rewrites each relative link in text, a file at rel, through resolve.
// Links with a scheme and links to an anchor on the same page stay.
func RewriteLinks(text, rel string, resolve Resolver) string {
	swap := func(target string) string {
		if url, ok := resolveLink(target, rel, resolve); ok {
			return url
		}
		return target
	}
	text = inlineLink.ReplaceAllStringFunc(text, func(m string) string {
		return "](" + swap(m[2:len(m)-1]) + ")"
	})
	return referenceLink.ReplaceAllStringFunc(text, func(m string) string {
		parts := referenceLink.FindStringSubmatch(m)
		return parts[1] + swap(parts[2])
	})
}

// resolveLink resolves one link target, relative to rel's folder, through
// resolve.
func resolveLink(target, rel string, resolve Resolver) (string, bool) {
	if strings.Contains(target, ":") || strings.HasPrefix(target, "#") {
		return "", false
	}
	file, anchor, _ := strings.Cut(target, "#")
	repoPath := path.Clean(path.Join(path.Dir(rel), file))
	if strings.HasPrefix(repoPath, "../") {
		return "", false
	}
	return resolve(repoPath, anchor)
}
