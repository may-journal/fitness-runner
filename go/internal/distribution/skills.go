package distribution

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// LinkSkills links every installed skill into each folder, rewriting stale links
// and leaving real folders and links someone else made alone.
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
	shipped := map[string]bool{}
	for _, skill := range skills {
		shipped[skill.Name()] = true
		target := filepath.Join(bundle, skillsFolder, skill.Name())
		if err := linkSkill(bundle, filepath.Join(folder, skill.Name()), target, log); err != nil {
			return err
		}
	}
	return removeDropped(bundle, folder, shipped, log)
}

// removeDropped deletes links in the folder that point into one of this tool's
// bundles for a skill the release no longer ships.
func removeDropped(bundle, folder string, shipped map[string]bool, log io.Writer) error {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		link := filepath.Join(folder, entry.Name())
		if err := removeIfDropped(bundle, link, shipped[entry.Name()], log); err != nil {
			return err
		}
	}
	return nil
}

func linkSkill(bundle, link, target string, log io.Writer) error {
	if reason := keepReason(bundle, link, target); reason != "" {
		fmt.Fprintf(log, "fitness: skill %s %s\n", link, reason)
		return nil
	}
	if err := replaceLink(link, target); err != nil {
		return err
	}
	fmt.Fprintf(log, "fitness: skill %s linked to %s\n", link, target)
	return nil
}

// keepReason says why a skill's link stays, or "" when it is missing, broken, or
// stale and gets written again.
func keepReason(bundle, link, target string) string {
	current, err := os.Readlink(link)
	if err != nil {
		return keepUnlinked(link)
	}
	if current == target {
		return "already linked"
	}
	if replaceable(bundle, link, current) {
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
// points into a bundle this tool unpacked.
func replaceable(bundle, link, current string) bool {
	if _, err := os.Stat(link); err != nil {
		return true
	}
	return ownedLink(bundle, current)
}

// ownedLink reports whether the target is a skill in the installed bundle's own
// cache, so another tool's bundle is never touched.
func ownedLink(bundle, current string) bool {
	skills := filepath.Dir(filepath.Clean(current))
	generation := filepath.Dir(skills)
	key := filepath.Dir(generation)
	return filepath.Base(skills) == skillsFolder &&
		generationPattern.MatchString(filepath.Base(generation)) &&
		strings.HasPrefix(filepath.Base(key), "fitness-") &&
		filepath.Dir(key) == filepath.Dir(filepath.Dir(filepath.Clean(bundle)))
}

var linkCounter atomic.Uint64

// replaceLink swaps the link in place so a reader never sees it missing. The
// temporary link has a name no other process or earlier run shares.
func replaceLink(link, target string) error {
	for {
		temporary := fmt.Sprintf("%s.new-%d-%s-%d", link, os.Getpid(), strconv.FormatInt(time.Now().UnixNano(), 36), linkCounter.Add(1))
		err := os.Symlink(target, temporary)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if err := os.Rename(temporary, link); err != nil {
			os.Remove(temporary)
			return err
		}
		return nil
	}
}

func removeIfDropped(bundle, link string, shipped bool, log io.Writer) error {
	current, err := os.Readlink(link)
	if shipped || err != nil || !ownedLink(bundle, current) {
		return nil
	}
	fmt.Fprintf(log, "fitness: skill %s removed, the release no longer ships it\n", link)
	return os.Remove(link)
}
