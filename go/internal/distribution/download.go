package distribution

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const maxArchiveBytes = 256 << 20
const maxMetadataBytes = 1 << 20

func (i *Installer) fetch(ctx context.Context, url string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	response, err := i.Client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: HTTP %d", url, response.StatusCode)
	}
	return readBounded(response.Body, limit)
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download exceeds %d bytes", limit)
	}
	return data, nil
}

func (i *Installer) resolveVersion(ctx context.Context, version string) (string, error) {
	if version != "latest" {
		return normalizeVersion(version)
	}
	data, err := i.fetch(ctx, i.LatestURL, maxMetadataBytes)
	if err != nil {
		return "", err
	}
	var release struct {
		Tag string `json:"tag_name"`
	}
	if err := json.Unmarshal(data, &release); err != nil {
		return "", err
	}
	fmt.Fprintf(i.Log, "fitness: latest resolved to %s\n", release.Tag)
	return normalizeVersion(release.Tag)
}

func (i *Installer) expectedHash(ctx context.Context, version, asset string) (string, error) {
	if version == i.Version && i.Hashes[i.Platform] != "" {
		return checkedHash(i.Hashes[i.Platform])
	}
	data, err := i.fetch(ctx, i.assetURL(version, checksumName), maxMetadataBytes)
	if err != nil {
		return "", err
	}
	return manifestHash(string(data), asset)
}

func manifestHash(manifest, asset string) (string, error) {
	var matches []string
	scanner := bufio.NewScanner(strings.NewReader(manifest))
	for scanner.Scan() {
		matches = appendMatch(matches, scanner.Text(), asset)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("expected one checksum for %s, got %d", asset, len(matches))
	}
	return checkedHash(matches[0])
}

func appendMatch(matches []string, line, asset string) []string {
	fields := strings.Fields(line)
	if len(fields) == 2 && fields[1] == asset {
		return append(matches, fields[0])
	}
	return matches
}

func checkedHash(hash string) (string, error) {
	if !hashPattern.MatchString(hash) {
		return "", fmt.Errorf("invalid SHA-256 hash %q", hash)
	}
	return hash, nil
}

func contentHash(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
func (i *Installer) assetURL(version, name string) string {
	return i.ReleaseURL + "/download/" + ReleaseTag(version) + "/" + name
}
func defaultClient() *http.Client { return &http.Client{Timeout: 2 * time.Minute} }

var legacyReleaseVersion = regexp.MustCompile(`^0\.[0-9]{8}\.[0-9]+$`)

// ReleaseTag keeps date-based releases on their historical module tag prefix.
func ReleaseTag(version string) string {
	if legacyReleaseVersion.MatchString(version) {
		return "go/v" + version
	}
	return "v" + version
}
