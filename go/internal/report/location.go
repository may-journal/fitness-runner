package report

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var sourceLocation = regexp.MustCompile(`^([^:\r\n]+):(?:(\d+)(?::\d+)?(?:\s|:|$)|\s)`)

func location(message string) string {
	match := sourceLocation.FindStringSubmatch(message)
	if match == nil {
		return ""
	}
	path := sourcePath(match[1])
	if path == "" {
		return ""
	}
	properties := " file=" + escapeProperty(path)
	if n, err := strconv.Atoi(match[2]); err == nil && n > 0 {
		properties += ",line=" + match[2]
	}
	return properties
}

func sourcePath(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if !plausiblePath(path) {
		return ""
	}
	workspace := os.Getenv("GITHUB_WORKSPACE")
	if workspace == "" {
		return relativePath(path)
	}
	absolute := resolvePath(workspace, path)
	relative, err := filepath.Rel(workspace, absolute)
	if err != nil {
		return ""
	}
	return relativePath(relative)
}

func plausiblePath(path string) bool {
	if info, err := os.Stat(path); err == nil {
		return !info.IsDir()
	}
	if workspace := os.Getenv("GITHUB_WORKSPACE"); workspace != "" {
		if info, err := os.Stat(filepath.Join(workspace, path)); err == nil {
			return !info.IsDir()
		}
	}
	return filepath.Ext(path) != "" || strings.ContainsAny(path, `/\\`)
}

func resolvePath(workspace, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	local := workingPath(workspace, path)
	if fileExists(local) {
		return local
	}
	candidate := filepath.Join(workspace, path)
	if fileExists(candidate) || local == "" {
		return candidate
	}
	return local
}

func workingPath(workspace, path string) string {
	current, err := os.Getwd()
	if err != nil {
		return ""
	}
	relative, err := filepath.Rel(workspace, current)
	if err != nil || relativePath(relative) == "" {
		return ""
	}
	return filepath.Join(current, path)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func relativePath(path string) string {
	if filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(path)
}
