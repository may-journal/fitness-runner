package release

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/report"
)

// Record is one release's cost on this host: asset sizes and install time
// from an empty cache.
type Record struct {
	Version   string        `json:"version"`
	Platform  string        `json:"platform"`
	InstallMs int64         `json:"installMs"`
	Assets    []AssetRecord `json:"assets"`
}

// AssetRecord is one release asset's size in bytes.
type AssetRecord struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

// Record measures the release in out, printing JSON to w and a size table to
// the GitHub Actions job summary.
func (c Config) Record(ctx context.Context, w io.Writer) error {
	metadata, err := c.Metadata()
	if err != nil {
		return err
	}
	record, err := c.measure(ctx, metadata)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	report.WriteSummary(recordMarkdown(record))
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func (c Config) measure(ctx context.Context, metadata Metadata) (Record, error) {
	record := Record{Version: metadata.Version, Platform: runtime.GOOS + "-" + runtime.GOARCH}
	for _, asset := range metadata.Assets {
		info, err := os.Stat(filepath.Join(c.Out, asset.Name))
		if err != nil {
			return record, err
		}
		record.Assets = append(record.Assets, AssetRecord{Name: asset.Name, Bytes: info.Size()})
	}
	ms, err := c.installTime(ctx, metadata)
	record.InstallMs = ms
	return record, err
}

// installTime times `fitness-install --install-only` against a cache that
// holds only the release archives, as a first install on a fresh runner.
func (c Config) installTime(ctx context.Context, metadata Metadata) (int64, error) {
	work, err := os.MkdirTemp("", "fitness-record-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(work)
	test, err := c.freshInstall(ctx, metadata, work)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	if err := test.command(ctx, test.installer, "--install-only").Run(); err != nil {
		return 0, fmt.Errorf("install-only run: %w", err)
	}
	return time.Since(start).Milliseconds(), nil
}

// freshInstall readies this host's installer in work with its own cache,
// seeded with the release archives so no download is timed.
func (c Config) freshInstall(ctx context.Context, metadata Metadata, work string) (smoke, error) {
	test, err := prepareSmokeWith(work, filepath.Join(work, "cache"))
	if err != nil {
		return test, err
	}
	if err := c.seedArchives(metadata, test.cache); err != nil {
		return test, err
	}
	return test, c.prepareInstaller(ctx, metadata, test.installer, false)
}

func recordMarkdown(r Record) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Release %s on %s\n\nInstall from an empty cache: %dms\n\n| Asset | Bytes |\n| --- | --- |\n", report.EscapeMarkdown(r.Version), r.Platform, r.InstallMs)
	for _, a := range r.Assets {
		fmt.Fprintf(&b, "| %s | %d |\n", report.EscapeMarkdown(a.Name), a.Bytes)
	}
	return b.String() + "\n"
}
