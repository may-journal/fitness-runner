package distribution

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
)

// checkTagsPath lists every check release tag under the API root, a page
// of at most 100 at a time.
const checkTagsPath = "/repos/may-journal/fitness-checks/git/matching-refs/tags/" + checkPrefix

// checkTagsName is the cache file recording the last tag lookup, so hooks run
// in quick succession ask GitHub once.
const checkTagsName = "check-tags.json"

// exactPin matches a pin naming one release, such as v1.2.3.
var exactPin = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// resolveByTags turns each pin into its release tag, in order. Exact pins
// need no lookup; a major takes its newest release from the check tags.
func (i *Installer) resolveByTags(ctx context.Context, pins []CheckPin) ([]checkRelease, error) {
	tags, err := i.tagsFor(ctx, pins)
	if err != nil {
		return nil, err
	}
	resolved := make([]checkRelease, len(pins))
	for k, pin := range pins {
		resolved[k].Tag = checkPrefix + pin.Name + "-" + pin.Pin
		if exactPin.MatchString(pin.Pin) {
			continue
		}
		if resolved[k].Tag = newestUnder(tags, pin.Name, pin.Pin); resolved[k].Tag == "" {
			return nil, fmt.Errorf("no release of check %s matches %s; see %s", pin.Name, pin.Pin, checksRoot)
		}
	}
	return resolved, nil
}

// tagsFor returns the check tags when a pin names a major, else nothing, so
// exact pins never ask GitHub.
func (i *Installer) tagsFor(ctx context.Context, pins []CheckPin) ([]string, error) {
	if !slices.ContainsFunc(pins, func(pin CheckPin) bool { return !exactPin.MatchString(pin.Pin) }) {
		return nil, nil
	}
	return i.checkTags(ctx)
}

// newestUnder is the newest release tag of check name within major, such as
// v1, by version, or "" when it has none.
func newestUnder(tags []string, name, major string) string {
	base := checkPrefix + name + "-v"
	newest, newestVersion := "", ""
	for _, tag := range tags {
		version := strings.TrimPrefix(tag, base)
		if strings.HasPrefix(tag, checkPrefix+name+"-"+major+".") && versionPattern.MatchString(version) && newer(version, newestVersion) {
			newest, newestVersion = tag, version
		}
	}
	return newest
}

// checkTags returns a fresh recorded tag lookup, else asks GitHub and records
// the answer.
func (i *Installer) checkTags(ctx context.Context) ([]string, error) {
	if tags, ok := readFresh[[]string](i.Cache, checkTagsName); ok {
		return tags, nil
	}
	tags, err := i.lookupTags(ctx)
	if err == nil {
		recordFresh(i.Cache, checkTagsName, tags)
	}
	return tags, err
}

// lookupTags reads every page of check release tags from the API.
func (i *Installer) lookupTags(ctx context.Context) ([]string, error) {
	var tags []string
	for page := 1; ; page++ {
		got, err := i.tagPage(ctx, page)
		if err != nil {
			return nil, err
		}
		tags = append(tags, got...)
		if len(got) < 100 {
			return tags, nil
		}
	}
}

// tagPage reads one page of check release tags, signed in when a GitHub
// token is at hand.
func (i *Installer) tagPage(ctx context.Context, page int) ([]string, error) {
	url := fmt.Sprintf("%s?per_page=100&page=%d", tagsURL(), page)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	i.announce(i.Lookup, "the check release tags", request)
	request.Header.Set("Accept", "application/vnd.github+json")
	if token := envToken(); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	data, err := read(i.Lookup, request, maxMetadataBytes)
	if err != nil {
		return nil, err
	}
	var refs []struct {
		Ref string `json:"ref"`
	}
	tags := []string{}
	err = json.Unmarshal(data, &refs)
	for _, ref := range refs {
		tags = append(tags, strings.TrimPrefix(ref.Ref, "refs/tags/"))
	}
	return tags, err
}

// tagsURL is the tag lookup endpoint under GITHUB_API_URL, which GitHub jobs
// set, else under the public API.
func tagsURL() string {
	root := strings.TrimRight(os.Getenv("GITHUB_API_URL"), "/")
	if root == "" {
		root = "https://api.github.com"
	}
	return root + checkTagsPath
}
