package distribution

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// bundleLink matches a link target inside an unpacked bundle's skills folder,
// such as one an earlier install made, which a new install may replace.
var bundleLink = regexp.MustCompile(`/bundle-[0-9]+/skills/[a-z0-9-]+$`)

// LinkSkills links every skill in the installed bundle into each folder, so an
// agent that reads the folder sees the skill. A missing or broken link, or one
// to an earlier bundle, is written again; a real folder or a link someone else
// made stays as it is.
func LinkSkills(bundle string, folders []string, log io.Writer) error {
	skills, err := os.ReadDir(filepath.Join(bundle, skillsFolder))
	if os.IsNotExist(err) {
		return fmt.Errorf("bundle %s ships no skills", bundle)
	}
	if err != nil {
		return err
	}
	for _, folder := range folders {
		if err := linkFolder(bundle, folder, skills, log); err != nil {
			return err
		}
	}
	return nil
}

func linkFolder(bundle, folder string, skills []os.DirEntry, log io.Writer) error {
	if err := os.MkdirAll(folder, 0755); err != nil {
		return err
	}
	for _, skill := range skills {
		target := filepath.Join(bundle, skillsFolder, skill.Name())
		if err := linkSkill(filepath.Join(folder, skill.Name()), target, log); err != nil {
			return err
		}
	}
	return nil
}

func linkSkill(link, target string, log io.Writer) error {
	if reason := keepReason(link, target); reason != "" {
		fmt.Fprintf(log, "fitness: skill %s %s\n", link, reason)
		return nil
	}
	if err := replaceLink(link, target); err != nil {
		return err
	}
	fmt.Fprintf(log, "fitness: skill %s linked to %s\n", link, target)
	return nil
}

// keepReason says why a skill's link stays as it is, or "" when it is missing,
// broken, or points into an earlier bundle and so gets written again.
func keepReason(link, target string) string {
	current, err := os.Readlink(link)
	if err != nil {
		return keepUnlinked(link)
	}
	if current == target {
		return "already linked"
	}
	if replaceable(link, current) {
		return ""
	}
	return "kept, linked to " + current
}

func keepUnlinked(link string) string {
	if _, err := os.Lstat(link); err == nil {
		return "kept, not a link"
	}
	return ""
}

// replaceable reports whether a link may be rewritten: it is broken or it
// points into a bundle.
func replaceable(link, current string) bool {
	if _, err := os.Stat(link); err != nil {
		return true
	}
	return bundleLink.MatchString(strings.TrimSuffix(current, "/"))
}

// replaceLink swaps the link in place so a reader never sees it missing.
func replaceLink(link, target string) error {
	temporary := link + ".new"
	os.Remove(temporary)
	if err := os.Symlink(target, temporary); err != nil {
		return err
	}
	return os.Rename(temporary, link)
}
