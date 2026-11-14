package distribution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// checkIndexURL is the release index fitness-checks republishes with every
// check release.
const checkIndexURL = checksRoot + "/download/fitness-check-index/index.json"

// checkIndexName is the cache file recording the last index read.
const checkIndexName = "check-index.json"

// errNoIndex says fitness-checks publishes no release index yet, so pins
// resolve through the check tags instead.
var errNoIndex = errors.New("fitness-checks publishes no release index")

// checkRelease is one check release to install. URL and SHA256 are empty
// when tags resolved it, and the release's own files give them.
type checkRelease struct {
	Tag, URL, SHA256 string
}

// checkIndex is the shape of the release index, format 1.
type checkIndex struct {
	Format int `json:"format"`
	Checks map[string]struct {
		Majors   map[string]string `json:"majors"`
		Versions map[string]struct {
			Tag       string `json:"tag"`
			Artifacts map[string]struct {
				URL    string `json:"url"`
				SHA256 string `json:"sha256"`
			} `json:"artifacts"`
		} `json:"versions"`
	} `json:"checks"`
}

// freshRecord is one answer from GitHub and when it came.
type freshRecord[T any] struct {
	Value   T         `json:"value"`
	Checked time.Time `json:"checked"`
}

// readFresh returns the answer recorded in the cache file name, and whether
// one younger than latestFresh was there.
func readFresh[T any](cache, name string) (T, bool) {
	var record freshRecord[T]
	data, err := readCacheFile(filepath.Join(cache, name), maxMetadataBytes)
	ok := err == nil && json.Unmarshal(data, &record) == nil && time.Since(record.Checked) < latestFresh
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

// resolvePins turns each pin into its release, in order: through the release
// index, else through the check tags while fitness-checks publishes none.
func (i *Installer) resolvePins(ctx context.Context, pins []CheckPin) ([]checkRelease, error) {
	index, err := i.readIndex(ctx)
	if err != nil {
		// Unanswered, the run asks nothing more, so only exact pins of cached
		// checks still install.
		i.Unanswered = i.Unanswered || !errors.Is(err, errNoIndex)
		return i.resolveByTags(ctx, pins)
	}
	releases := make([]checkRelease, len(pins))
	for k, pin := range pins {
		if releases[k], err = index.release(pin, i.Platform); err != nil {
			return nil, err
		}
	}
	return releases, nil
}

// release finds pin's artifact for platform: an exact version, or the
// version the index names for a major.
func (index checkIndex) release(pin CheckPin, platform string) (checkRelease, error) {
	check := index.Checks[pin.Name]
	version := strings.TrimPrefix(pin.Pin, "v")
	if !exactPin.MatchString(pin.Pin) {
		version = check.Majors[version]
	}
	artifact, ok := check.Versions[version].Artifacts[platform]
	if !ok || check.Versions[version].Tag == "" {
		return checkRelease{}, fmt.Errorf("no release of check %s matches %s for %s; see %s", pin.Name, pin.Pin, platform, checksRoot)
	}
	return checkRelease{Tag: check.Versions[version].Tag, URL: artifact.URL, SHA256: artifact.SHA256}, nil
}

// checkLookupTimeout bounds each read of the check index or tags, like the
// runner's GitHub probe. A silent network then delays a hook little.
const checkLookupTimeout = 2 * time.Second

// checkLookup is the lookup client with checkLookupTimeout.
func (i *Installer) checkLookup() *http.Client {
	return &http.Client{Timeout: checkLookupTimeout, Transport: i.Lookup.Transport}
}

// readIndex returns a fresh recorded index, else reads and records it.
// errNoIndex means fitness-checks publishes none.
func (i *Installer) readIndex(ctx context.Context) (checkIndex, error) {
	if index, ok := readFresh[checkIndex](i.Cache, checkIndexName); ok {
		return index, publishedIndex(index)
	}
	index, err := i.fetchIndex(ctx)
	if err == nil || errors.Is(err, errNoIndex) {
		// A missing index is recorded as format 0, so runs ask again later.
		recordFresh(i.Cache, checkIndexName, index)
	}
	return index, err
}

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
		return index, fmt.Errorf("download %s: %w", checkIndexURL, err)
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
