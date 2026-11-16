package distribution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
)

// checkIndexURL is the release index fitness-checks republishes with every
// check release.
const checkIndexURL = checksRoot + "/download/fitness-check-index/index.json"

// checkIndexName is the cache file recording the last index read.
const checkIndexName = "check-index.json"

// errNoIndex says fitness-checks publishes no release index yet, so pins
// resolve through the check tags instead.
var errNoIndex = errors.New("fitness-checks publishes no release index")

// errUnanswered marks an index read that got no answer from the network.
var errUnanswered = errors.New("GitHub did not answer the check release index")

// checkRelease is one check release to install. URL and SHA256 are empty
// when tags resolved it, and the release's own files give them.
type checkRelease struct {
	Tag, URL, SHA256 string
}

// checkIndex is the shape of the release index, format 1.
type checkIndex struct {
	Format int `json:"format"`
	Checks map[string]struct {
		Majors   map[string]string       `json:"majors"`
		Versions map[string]checkVersion `json:"versions"`
	} `json:"checks"`
}

// checkVersion is one check version in the index: its tag, its artifact
// per platform, and the fields its --describe declares. A version listed
// before checks declared fields has none, so every file reaches it.
type checkVersion struct {
	Tag       string `json:"tag"`
	Artifacts map[string]struct {
		URL    string `json:"url"`
		SHA256 string `json:"sha256"`
	} `json:"artifacts"`
	Files    string                 `json:"files"`
	Exclude  string                 `json:"exclude"`
	Needs    string                 `json:"needs"`
	Requires []checkkit.Requirement `json:"requires"`
}

// freshRecord is one answer from GitHub and when it came.
type freshRecord[T any] struct {
	Value   T         `json:"value"`
	Checked time.Time `json:"checked"`
}

// readFresh returns the answer recorded in the cache file name, and whether
// one younger than within was there.
func readFresh[T any](cache, name string, within time.Duration) (T, bool) {
	var record freshRecord[T]
	data, err := readCacheFile(filepath.Join(cache, name), maxMetadataBytes)
	ok := err == nil && json.Unmarshal(data, &record) == nil && time.Since(record.Checked) < within
	return record.Value, ok
}

// recordFresh records an answer in the cache file name. A record that cannot
// be written only means the next run asks again.
func recordFresh[T any](cache, name string, value T) {
	data, err := json.Marshal(freshRecord[T]{Value: value, Checked: time.Now()})
	if err == nil && os.MkdirAll(cache, 0700) == nil {
		_ = writeAtomic(filepath.Join(cache, name), data)
	}
}

// ReleasedChecks names every check the fitness-checks release index lists,
// in name order. Only the index lists them all, so a read that fails says
// to pin each check instead.
func (i *Installer) ReleasedChecks(ctx context.Context) ([]string, error) {
	index, err := i.readIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the fitness-checks release index: %w; pin each check in checkVersions instead", err)
	}
	return slices.Sorted(maps.Keys(index.Checks)), nil
}

// resolvePins turns each pin into its release, in order: through the release
// index, else through the check tags while fitness-checks publishes none.
func (i *Installer) resolvePins(ctx context.Context, pins []CheckPin) ([]checkRelease, error) {
	index, err := i.readIndex(ctx)
	// Unanswered, the run asks nothing more, so only exact pins of cached
	// checks still install.
	i.Unanswered = i.Unanswered || errors.Is(err, errUnanswered)
	if errors.Is(err, errNoIndex) || errors.Is(err, errUnanswered) {
		return i.resolveByTags(ctx, pins)
	}
	if err != nil {
		return nil, err
	}
	return index.releases(pins, i.Platform), nil
}

// releases resolves each pin through the index, in order.
func (index checkIndex) releases(pins []CheckPin, platform string) []checkRelease {
	releases := make([]checkRelease, len(pins))
	for k, pin := range pins {
		releases[k] = index.release(pin, platform)
	}
	return releases
}

// release finds pin's artifact for platform: an exact version, or the
// version the index names for a major. An exact pin the index lacks takes
// its tag's own files; a major it lacks resolves to nothing.
func (index checkIndex) release(pin CheckPin, platform string) checkRelease {
	version := index.version(pin)
	artifact, ok := version.Artifacts[platform]
	switch {
	case ok && version.Tag != "":
		return checkRelease{Tag: version.Tag, URL: artifact.URL, SHA256: artifact.SHA256}
	case exactPin.MatchString(pin.Pin):
		return checkRelease{Tag: checkPrefix + pin.Name + "-" + pin.Pin}
	}
	return checkRelease{}
}

// version is the index entry pin names: an exact version, or the version
// the index names for a major. A pin the index lacks gets the zero entry.
func (index checkIndex) version(pin CheckPin) checkVersion {
	check := index.Checks[pin.Name]
	version := strings.TrimPrefix(pin.Pin, "v")
	if !exactPin.MatchString(pin.Pin) {
		version = check.Majors[version]
	}
	return check.Versions[version]
}

// checkLookupTimeout bounds each read of the check index or tags, like the
// runner's GitHub probe. A silent network then delays a hook little.
const checkLookupTimeout = 2 * time.Second

// checkLookup is the lookup client with checkLookupTimeout.
func (i *Installer) checkLookup() *http.Client {
	return &http.Client{Timeout: checkLookupTimeout, Transport: i.Lookup.Transport}
}

// readIndex returns a fresh recorded index, else reads and records it, or
// offline the last one recorded; errNoIndex means fitness-checks has none.
func (i *Installer) readIndex(ctx context.Context) (checkIndex, error) {
	if index, ok := readFresh[checkIndex](i.Cache, checkIndexName, latestFresh); ok {
		return index, publishedIndex(index)
	}
	if _, ok := readFresh[bool](i.Cache, unansweredName, unansweredFresh); ok {
		return i.offlineIndex(errUnanswered, false)
	}
	index, err := i.fetchIndex(ctx)
	i.recordIndex(index, err)
	if errors.Is(err, errUnanswered) {
		return i.offlineIndex(err, true)
	}
	return index, err
}

// offlineIndex is the last recorded index, however old, or err without one.
// The run then asks GitHub nothing more, and warn names the fallback.
func (i *Installer) offlineIndex(err error, warn bool) (checkIndex, error) {
	var record freshRecord[checkIndex]
	data, readErr := readCacheFile(filepath.Join(i.Cache, checkIndexName), maxMetadataBytes)
	if readErr != nil || json.Unmarshal(data, &record) != nil || record.Value.Format != 1 {
		return checkIndex{}, err
	}
	i.Unanswered = true
	if warn {
		fmt.Fprintf(i.Log, "fitness: warning: %v, so this run is offline and matches against the index cached %s\n", err, record.Checked.Format(time.DateTime))
	}
	return record.Value, nil
}

// recordIndex records an index read. A missing index is format 0, so runs
// ask again later; an unanswered read has its own record.
func (i *Installer) recordIndex(index checkIndex, err error) {
	switch {
	case err == nil || errors.Is(err, errNoIndex):
		recordFresh(i.Cache, checkIndexName, index)
	case errors.Is(err, errUnanswered):
		recordFresh(i.Cache, unansweredName, true)
	}
}

// unansweredName records an index read that got no answer, so the next hook
// of the same commit does not wait for it again.
const unansweredName = "check-index-unanswered.json"

// unansweredFresh is how long an unanswered index read stands.
const unansweredFresh = time.Minute

// publishedIndex is errNoIndex for a recorded missing index, else nil.
func publishedIndex(index checkIndex) error {
	if index.Format == 0 {
		return errNoIndex
	}
	return nil
}

// fetchIndex reads the release index, refusing a format it does not know.
// A missing index is errNoIndex.
func (i *Installer) fetchIndex(ctx context.Context) (checkIndex, error) {
	var index checkIndex
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, checkIndexURL, nil)
	if err != nil {
		return index, err
	}
	client := i.checkLookup()
	i.announce(client, "the check release index", request)
	response, err := client.Do(request)
	if err != nil {
		return index, fmt.Errorf("%w: %w", errUnanswered, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return index, errNoIndex
	}
	if response.StatusCode != http.StatusOK {
		return index, fmt.Errorf("download %s: HTTP %d", checkIndexURL, response.StatusCode)
	}
	return decodeIndex(response)
}

// decodeIndex reads a format 1 index from response.
func decodeIndex(response *http.Response) (checkIndex, error) {
	var index checkIndex
	data, err := readBounded(response.Body, maxMetadataBytes)
	if err == nil {
		err = json.Unmarshal(data, &index)
	}
	if err == nil && index.Format != 1 {
		err = fmt.Errorf("the check release index has format %d; this fitness-install reads format 1, so update it", index.Format)
	}
	return index, err
}
