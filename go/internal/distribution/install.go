package distribution

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// Installer holds typed release, transport, and cache settings. The command
// uses public HTTPS endpoints; tests inject a local HTTP server.
type Installer struct {
	Version, Platform, Cache, ReleaseURL, LatestURL string
	Hashes                                          map[string]string
	Client                                          *http.Client
	Log                                             io.Writer
}

func New(version, hashes string, log io.Writer) (*Installer, error) {
	platform, err := Platform(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}
	cache, err := cacheDirectory()
	if err != nil {
		return nil, err
	}
	return &Installer{Version: version, Hashes: ParseHashes(hashes), Platform: platform, Cache: cache, ReleaseURL: releaseRoot, LatestURL: latestRelease, Client: defaultClient(), Log: log}, nil
}

func (i *Installer) Install(ctx context.Context, requested string) (string, error) {
	version, err := i.resolveVersion(ctx, requested)
	if err != nil {
		return "", err
	}
	top := "fitness-" + version + "-" + i.Platform
	digest, err := i.expectedHash(ctx, version, top+".tar.gz")
	if err != nil {
		return "", err
	}
	fmt.Fprintf(i.Log, "fitness: version=%s platform=%s sha256=%s\n", version, i.Platform, digest)
	return i.installBundle(ctx, version, top, digest)
}

func (i *Installer) installBundle(ctx context.Context, version, top, digest string) (string, error) {
	key := filepath.Join(i.Cache, top+"-"+digest)
	if path := recordedBundle(key); path != "" {
		return path, nil
	}
	if err := os.MkdirAll(key, 0700); err != nil {
		return "", err
	}
	data, err := i.archive(ctx, key, version, top+".tar.gz", digest)
	if err != nil {
		return "", err
	}
	return stageBundle(key, top, data)
}

func (i *Installer) archive(ctx context.Context, key, version, asset, digest string) ([]byte, error) {
	path := filepath.Join(key, "archive.tar.gz")
	if data, valid := cachedArchive(path, digest); valid {
		return data, nil
	}
	data, err := i.fetch(ctx, i.assetURL(version, asset), maxArchiveBytes)
	if err != nil {
		return nil, err
	}
	if contentHash(data) != digest {
		return nil, fmt.Errorf("checksum mismatch for %s", asset)
	}
	if err := writeAtomic(path, data); err != nil {
		return nil, err
	}
	return data, nil
}

func stageBundle(key, top string, data []byte) (string, error) {
	staged, err := os.MkdirTemp(key, "bundle-")
	if err != nil {
		return "", err
	}
	path, err := chooseBundle(key, staged, top, data)
	if path != staged {
		os.RemoveAll(staged)
	}
	return path, err
}

func chooseBundle(key, staged, top string, data []byte) (string, error) {
	hashes, err := extractArchive(data, staged, top)
	if err != nil {
		return "", err
	}
	if cached := currentBundle(key, hashes); cached != "" {
		// A bundle installed before hashes were recorded gets its record now.
		_ = recordHashes(key, filepath.Base(cached), hashes)
		return cached, nil
	}
	if err := recordHashes(key, filepath.Base(staged), hashes); err != nil {
		return "", err
	}
	return publishBundle(key, staged)
}
