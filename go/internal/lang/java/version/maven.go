package version

// cspell:ignore Dmaven

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
	// repoLocalRe reads the local repository a Maven command line sets.
	repoLocalRe = regexp.MustCompile(`-Dmaven\.repo\.local=(\S+)`)
	// settingsLocalRe reads the local repository Maven's settings set.
	settingsLocalRe = regexp.MustCompile(`<localRepository>\s*([^<]*?)\s*</localRepository>`)
)

// maxParents bounds a parent chain, so a pom that names itself ends.
const maxParents = 16

// parent is a pom's parent: its coordinates and the relative path to its pom,
// empty when Maven uses only a repository.
type parent struct {
	group, artifact, version, relative string
}

// chain is the poms a pom inherits from, nearest first, itself included.
// The first pins of them name the pom's own release: itself and the
// parents outside the repo above it. A parent in the repo names its own,
// so it and those above it lend only their properties.
type chain struct {
	poms []string
	pins int
	// unread is why a parent that would name the release went unread.
	unread string
}

// mavenPins reads the releases a pom names, or else those its parents
// outside the repo name, and why such a parent went unread.
func mavenPins(root, file string, _ []java.Project) ([]pin, string) {
	body, ok := read(root, file)
	if !ok {
		return nil, ""
	}
	c := inherited(root, file, body)
	for _, pom := range c.poms[:c.pins] {
		if found := releases(pom, c.poms); len(found) > 0 {
			return filePins(file, found), ""
		}
	}
	return nil, c.unread
}

// filePins pins each version to file.
func filePins(file string, versions []string) []pin {
	out := make([]pin, 0, len(versions))
	for _, v := range versions {
		out = append(out, pin{file: file, version: v})
	}
	return out
}

// inherited reads the chain of the pom at file, each parent from the repo if
// found there, else from the local Maven repository.
func inherited(root, file, body string) chain {
	c := chain{poms: []string{body}, pins: 1}
	at, own := file, true
	for p, ok := parentOf(body); ok && len(c.poms) < maxParents; p, ok = parentOf(body) {
		var found bool
		if body, at, own, found = parentPom(root, at, own, p); !found {
			c.unread = unread(own, file, p)
			return c
		}
		c.poms = append(c.poms, body)
		if own {
			c.pins++
		}
	}
	return c
}

// parentPom reads p, named by the pom at at, from the repo, else from
// the local Maven repository.
func parentPom(root, at string, own bool, p parent) (body, next string, stillOwn, found bool) {
	if rel, in := repoParent(root, at, p); in {
		return readOr(root, rel), rel, false, true
	}
	pom, err := os.ReadFile(localPom(root, p))
	return string(pom), "", own, err == nil
}

// readOr returns a tracked file's text, or empty when it is missing.
func readOr(root, file string) string {
	body, _ := read(root, file)
	return body
}

// unread is why a pom's own parent went unread, when that parent would
// name its release.
func unread(own bool, file string, p parent) string {
	if !own {
		return ""
	}
	return fmt.Sprintf("%s inherits %s:%s:%s, which the local Maven repository lacks, so its release is unread; run the build once to fetch it", file, p.group, p.artifact, p.version)
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

// repoParent finds the parent's pom in the repo as Maven does, by
// relative path and artifact.
func repoParent(root, file string, p parent) (string, bool) {
	rel, ok := relativePom(file, p)
	if !ok {
		return "", false
	}
	body, ok := read(root, rel)
	return rel, ok && tag(parentRe.ReplaceAllString(body, ""), "artifactId") == p.artifact
}

// relativePom is the repo path p names from the pom at file, or false when
// it leaves the repo.
func relativePom(file string, p parent) (string, bool) {
	if file == "" || p.relative == "" {
		return "", false
	}
	rel := path.Join(path.Dir(file), p.relative)
	if !strings.HasSuffix(rel, ".xml") {
		rel = path.Join(rel, "pom.xml")
	}
	return rel, rel != ".." && !strings.HasPrefix(rel, "../")
}

// localPom is where the local Maven repository keeps a parent's pom.
func localPom(root string, p parent) string {
	return filepath.Join(localRepository(root), filepath.FromSlash(strings.ReplaceAll(p.group, ".", "/")), p.artifact, p.version, p.artifact+"-"+p.version+".pom")
}

// localRepository is the one the repo's maven.config, MAVEN_OPTS or the user
// settings set, else ~/.m2/repository.
func localRepository(root string) string {
	home, _ := os.UserHomeDir()
	config := readOr(root, ".mvn/maven.config") + " " + os.Getenv("MAVEN_OPTS")
	if m := repoLocalRe.FindStringSubmatch(config); m != nil && filepath.IsAbs(m[1]) {
		return m[1]
	} else if m != nil {
		return filepath.Join(root, m[1])
	}
	settings, _ := os.ReadFile(filepath.Join(home, ".m2", "settings.xml"))
	if m := settingsLocalRe.FindStringSubmatch(string(settings)); m != nil {
		return strings.ReplaceAll(m[1], "${user.home}", home)
	}
	return filepath.Join(home, ".m2", "repository")
}
