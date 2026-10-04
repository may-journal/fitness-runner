package release

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
)

type smoke struct {
	work, tools, cache, repo, installer string
	env                                 []string
}

func (c Config) Smoke(ctx context.Context, published bool) error {
	metadata, err := c.Metadata()
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "fitness-release-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	return c.smokeAt(ctx, metadata, work, published)
}

func (c Config) smokeAt(ctx context.Context, metadata Metadata, work string, published bool) error {
	test, err := prepareSmoke(work)
	if err != nil {
		return err
	}
	if !published {
		if err := c.seedArchives(metadata, test.cache); err != nil {
			return err
		}
	}
	if err := c.prepareInstaller(ctx, metadata, test.installer, published); err != nil {
		return err
	}
	return test.check(ctx)
}

func (c Config) prepareInstaller(ctx context.Context, metadata Metadata, path string, published bool) error {
	name := executable(installerName, metadata.Version, runtime.GOOS+"-"+runtime.GOARCH)
	data, err := c.installerData(ctx, metadata.Version, name, published)
	if err != nil {
		return err
	}
	if err := verifyAsset(metadata, name, data); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0700)
}

func (c Config) installerData(ctx context.Context, version, name string, published bool) ([]byte, error) {
	if !published {
		return os.ReadFile(filepath.Join(c.Out, name))
	}
	url := "https://github.com/may-journal/fitness-runner/releases/download/" + distribution.ReleaseTag(version) + "/" + name
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return readResponse(response)
}

func readResponse(response *http.Response) ([]byte, error) {
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("installer download: HTTP %d", response.StatusCode)
	}
	return io.ReadAll(io.LimitReader(response.Body, 64<<20))
}

func verifyAsset(metadata Metadata, name string, data []byte) error {
	for _, asset := range metadata.Assets {
		if asset.Name == name {
			return verifyHash(asset.Hash, data)
		}
	}
	return fmt.Errorf("asset %s is absent from release metadata", name)
}

func (c Config) seedArchives(metadata Metadata, cache string) error {
	for _, asset := range metadata.Assets {
		if !strings.HasSuffix(asset.Name, ".tar.gz") {
			continue
		}
		if err := c.seedArchive(asset, cache); err != nil {
			return err
		}
	}
	return nil
}

func (c Config) seedArchive(asset Asset, cache string) error {
	key := filepath.Join(cache, strings.TrimSuffix(asset.Name, ".tar.gz")+"-"+asset.Hash)
	if err := os.MkdirAll(key, 0700); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(c.Out, asset.Name))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(key, "archive.tar.gz"), data, 0600)
}
