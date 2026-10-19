package distribution

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// latestName is the cache file recording the last answer to the latest
// release lookup, so hooks run in quick succession ask GitHub once.
const latestName = "latest-release.json"

// latestFresh is how long a recorded answer stands without asking again.
const latestFresh = 10 * time.Minute

// latestPath is the releases API path, appended to the API root.
const latestPath = "/repos/may-journal/fitness-runner/releases/latest"

// latestRecord is one answer to the latest release lookup and when it came.
type latestRecord struct {
	Tag     string    `json:"tag"`
	Checked time.Time `json:"checked"`
}

// latestURL is the releases API endpoint under GITHUB_API_URL, which GitHub
// jobs set, or under the public API.
func latestURL() string {
	if root := strings.TrimRight(os.Getenv("GITHUB_API_URL"), "/"); root != "" {
		return root + latestPath
	}
	return latestRelease
}

// resolveLatest answers from a fresh recorded lookup, else asks GitHub and
// records the answer. When GitHub refuses, it falls back to the recorded or
// newest installed version with one warning.
func (i *Installer) resolveLatest(ctx context.Context) (string, error) {
	record := i.readLatest()
	if record.Tag != "" && time.Since(record.Checked) < latestFresh {
		fmt.Fprintf(i.Log, "fitness: latest resolved to %s (checked %s ago)\n", record.Tag, time.Since(record.Checked).Round(time.Second))
		return normalizeVersion(record.Tag)
	}
	tag, err := i.lookupLatest(ctx)
	if err != nil {
		return i.fallBack(err, record.Tag)
	}
	fmt.Fprintf(i.Log, "fitness: latest resolved to %s\n", tag)
	i.writeLatest(tag)
	return normalizeVersion(tag)
}

// fallBack answers a failed lookup with the recorded tag, else the newest
// installed version, and one warning naming the failure. With neither, the
// failure stands.
func (i *Installer) fallBack(failure error, recorded string) (string, error) {
	if recorded == "" {
		recorded = i.newestInstalled()
	}
	if recorded == "" {
		return "", failure
	}
	version, err := normalizeVersion(recorded)
	fmt.Fprintf(i.Log, "fitness: warning: %v; using %s\n", failure, version)
	return version, err
}

// lookupLatest asks the releases API for the newest release tag, signed in
// when a GitHub token is at hand.
func (i *Installer) lookupLatest(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, i.LatestURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	if token := githubToken(ctx); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	data, err := i.send(request, maxMetadataBytes)
	if err != nil {
		return "", err
	}
	var release struct {
		Tag string `json:"tag_name"`
	}
	if err := json.Unmarshal(data, &release); err != nil {
		return "", err
	}
	_, err = normalizeVersion(release.Tag)
	return release.Tag, err
}

// githubToken is GH_TOKEN, else GITHUB_TOKEN, else what a logged-in gh
// prints, or "" when none is at hand.
func githubToken(ctx context.Context) string {
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gh", "auth", "token", "--hostname", "github.com").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// readLatest returns the recorded lookup, or an empty record when none is
// recorded or the record names no valid version.
func (i *Installer) readLatest() latestRecord {
	var record latestRecord
	data, err := readCacheFile(filepath.Join(i.Cache, latestName), maxMetadataBytes)
	if err != nil || json.Unmarshal(data, &record) != nil {
		return latestRecord{}
	}
	if _, err := normalizeVersion(record.Tag); err != nil {
		return latestRecord{}
	}
	return record
}

// writeLatest records a lookup's answer. A record that cannot be written
// only means the next run asks GitHub again.
func (i *Installer) writeLatest(tag string) {
	data, err := json.Marshal(latestRecord{Tag: tag, Checked: time.Now()})
	if err != nil || os.MkdirAll(i.Cache, 0700) != nil {
		return
	}
	_ = writeAtomic(filepath.Join(i.Cache, latestName), data)
}

// newestInstalled is the highest version with an intact installed bundle
// for this platform, or "" when none is installed.
func (i *Installer) newestInstalled() string {
	entries, err := os.ReadDir(i.Cache)
	if err != nil {
		return ""
	}
	pattern := regexp.MustCompile(`^fitness-([0-9]+\.[0-9]+\.[0-9]+)-` + regexp.QuoteMeta(i.Platform) + `-[a-f0-9]{64}$`)
	newest := ""
	for _, entry := range entries {
		if version := i.installed(pattern, entry.Name()); newer(version, newest) {
			newest = version
		}
	}
	return newest
}

// installed is the version a cache entry holds intact, or "" when the entry
// is no bundle for this platform or a file in it is damaged.
func (i *Installer) installed(pattern *regexp.Regexp, name string) string {
	match := pattern.FindStringSubmatch(name)
	if match == nil || recordedBundle(filepath.Join(i.Cache, name)) == "" {
		return ""
	}
	return match[1]
}

// newer reports whether version a is above b. Any version is above "", and
// "" is above none.
func newer(a, b string) bool {
	if a == "" || b == "" {
		return a != ""
	}
	left, right := strings.Split(a, "."), strings.Split(b, ".")
	for part := range left {
		x, _ := strconv.Atoi(left[part])
		y, _ := strconv.Atoi(right[part])
		if x != y {
			return x > y
		}
	}
	return false
}
