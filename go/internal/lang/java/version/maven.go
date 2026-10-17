package version

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

var (
	// parentRe reads the parent block a pom inherits from.
	parentRe = regexp.MustCompile(`(?s)<parent>(.*?)</parent>`)
	// relativeRe reads a parent's relativePath, empty when the tag is
	// self-closed.
	relativeRe = regexp.MustCompile(`<relativePath\s*/>|<relativePath>\s*([^<]*?)\s*</relativePath>`)
)

// parent is a pom's parent: its coordinates, and the path to its pom in
// the repo, empty when Maven looks it up only in a repository.
type parent struct {
	group, artifact, version, relative string
}

// mavenPins reads the releases a pom names, or else those its parents
// outside the repo name, and why such a parent went unread.
func mavenPins(root, file string, _ []java.Project) ([]pin, string) {
	body, ok := read(root, file)
	if !ok {
		return nil, ""
	}
	poms, why := inherited(root, file, body)
	for _, pom := range poms {
		if found := releases(pom, poms); len(found) > 0 {
			return filePins(file, found), ""
		}
	}
	return nil, why
}

// filePins pins each version to file.
func filePins(file string, versions []string) []pin {
	out := make([]pin, 0, len(versions))
	for _, v := range versions {
		out = append(out, pin{file: file, version: v})
	}
	return out
}

// inherited lists the pom body and, when its parent lives outside the
// repo, each parent pom the local Maven repository holds, nearest first,
// and the parent it lacks, if any.
func inherited(root, file, body string) ([]string, string) {
	poms := []string{body}
	p, ok := parentOf(body)
	if !ok || inRepo(root, file, p) {
		return poms, ""
	}
	for ok {
		pom, err := os.ReadFile(localPom(p))
		if err != nil {
			return poms, fmt.Sprintf("%s inherits %s:%s:%s, which the local Maven repository lacks, so its release is unread; run the build once to fetch it", file, p.group, p.artifact, p.version)
		}
		poms = append(poms, string(pom))
		p, ok = parentOf(string(pom))
	}
	return poms, ""
}

// parentOf reads the parent a pom names, with Maven's default relative
// path of ../pom.xml.
func parentOf(body string) (parent, bool) {
	m := parentRe.FindStringSubmatch(body)
	if m == nil {
		return parent{}, false
	}
	p := parent{group: tag(m[1], "groupId"), artifact: tag(m[1], "artifactId"), version: tag(m[1], "version"), relative: "../pom.xml"}
	if r := relativeRe.FindStringSubmatch(m[1]); r != nil {
		p.relative = r[1]
	}
	return p, p.group != "" && p.artifact != "" && p.version != ""
}

// tag reads the text of a child element of a block.
func tag(block, name string) string {
	m := regexp.MustCompile(`<` + name + `>\s*([^<\s]+)\s*</` + name + `>`).FindStringSubmatch(block)
	if m == nil {
		return ""
	}
	return m[1]
}

// inRepo reports a parent whose relative path, from the pom's folder,
// names a pom in the repo, which the check reads on its own.
func inRepo(root, file string, p parent) bool {
	if p.relative == "" {
		return false
	}
	rel := path.Join(path.Dir(file), p.relative)
	if !strings.HasSuffix(rel, ".xml") {
		rel = path.Join(rel, "pom.xml")
	}
	_, ok := read(root, rel)
	return ok
}

// localPom is where the default local Maven repository keeps a parent's
// pom.
func localPom(p parent) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".m2", "repository", filepath.FromSlash(strings.ReplaceAll(p.group, ".", "/")), p.artifact, p.version, p.artifact+"-"+p.version+".pom")
}
