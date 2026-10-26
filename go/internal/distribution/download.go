package distribution

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const maxArchiveBytes = 256 << 20
const maxMetadataBytes = 1 << 20

// lookupTimeout bounds the latest release lookup and the hash manifest, so a
// network that drops traffic delays a commit by seconds, not minutes.
const lookupTimeout = 5 * time.Second

// announce prints the host a read is about to ask and how long it waits, so
// a wait on the network is never silent.
func (i *Installer) announce(client *http.Client, what string, request *http.Request) {
	fmt.Fprintf(i.Log, "fitness: asking %s for %s (gives up after %s)\n", request.URL.Host, what, client.Timeout)
}

// fetch reads the small file at url through the lookup client.
func (i *Installer) fetch(ctx context.Context, what, url string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	i.announce(i.Lookup, what, request)
	return read(i.Lookup, request, limit)
}

// read sends request through client and reads at most limit bytes of an
// HTTP 200 answer.
func read(client *http.Client, request *http.Request, limit int64) ([]byte, error) {
	response, err := open(client, request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return readBounded(response.Body, limit)
}

// download reads the bundle asset at url through the two-minute client,
// printing its size before the body is read.
func (i *Installer) download(ctx context.Context, url, asset string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	i.announce(i.Client, asset, request)
	response, err := open(i.Client, request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	fmt.Fprintf(i.Log, "fitness: downloading %s (%s)\n", asset, size(response.ContentLength))
	return readBounded(response.Body, maxArchiveBytes)
}

// open sends request through client and returns a response that answered
// HTTP 200, or an error naming the URL.
func open(client *http.Client, request *http.Request) (*http.Response, error) {
	url := request.URL.String()
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("download %s: HTTP %d", url, response.StatusCode)
	}
	return response, nil
}

// size is a byte count in megabytes, or "size unknown" when the server sent
// no length.
func size(bytes int64) string {
	if bytes < 0 {
		return "size unknown"
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1<<20))
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
	return i.resolveLatest(ctx)
}

func (i *Installer) expectedHash(ctx context.Context, version, asset string) (string, error) {
	if version == i.Version && i.Hashes[i.Platform] != "" {
		return checkedHash(i.Hashes[i.Platform])
	}
	data, err := i.fetch(ctx, checksumName, i.assetURL(version, checksumName), maxMetadataBytes)
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
func lookupClient() *http.Client  { return &http.Client{Timeout: lookupTimeout} }

var legacyReleaseVersion = regexp.MustCompile(`^0\.[0-9]{8}\.[0-9]+$`)

// ReleaseTag keeps date-based releases on their historical module tag prefix.
func ReleaseTag(version string) string {
	if legacyReleaseVersion.MatchString(version) {
		return "go/v" + version
	}
	return "v" + version
}
