// cspell:ignore Typeflag
package distribution

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxBinaryBytes = 64 << 20
const maxBundleBytes = 1 << 30

type extractedBundle struct {
	Hashes map[string]string
	Bytes  int64
}

func extractArchive(data []byte, directory, top string) (map[string]string, error) {
	compressed, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	bundle := extractedBundle{Hashes: map[string]string{}}
	if err := bundle.extract(reader, directory, top); err != nil {
		return nil, err
	}
	// Finish reading gzip so a bad trailer cannot escape validation.
	if _, err := io.Copy(io.Discard, compressed); err != nil {
		return nil, err
	}
	if bundle.Hashes["fitness"] == "" {
		return nil, errors.New("archive lacks the fitness runner")
	}
	return bundle.Hashes, nil
}

func (b *extractedBundle) extract(reader *tar.Reader, directory, top string) error {
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := b.extractEntry(reader, header, directory, top); err != nil {
			return err
		}
	}
}

func (b *extractedBundle) extractEntry(reader io.Reader, h *tar.Header, directory, top string) error {
	if h.Typeflag == tar.TypeDir && h.Name == top+"/" {
		return nil
	}
	name, err := archiveName(h, top)
	if err != nil {
		return err
	}
	if _, exists := b.Hashes[name]; exists {
		return fmt.Errorf("duplicate archive file %s", name)
	}
	return b.writeBinary(reader, h, directory, name)
}

func archiveName(h *tar.Header, top string) (string, error) {
	if h.Typeflag != tar.TypeReg {
		return "", fmt.Errorf("archive entry %q is not a regular binary", h.Name)
	}
	name := strings.TrimPrefix(h.Name, top+"/")
	if name == h.Name || !binaryPattern.MatchString(name) {
		return "", fmt.Errorf("unsafe archive path %q", h.Name)
	}
	if h.Mode&0111 == 0 {
		return "", fmt.Errorf("archive binary %s is not executable", name)
	}
	return name, nil
}

func (b *extractedBundle) writeBinary(reader io.Reader, h *tar.Header, directory, name string) error {
	if h.Size < 0 || h.Size > maxBinaryBytes {
		return fmt.Errorf("invalid binary size for %s", name)
	}
	b.Bytes += h.Size
	if b.Bytes > maxBundleBytes {
		return errors.New("expanded bundle exceeds size limit")
	}
	data, err := readBounded(reader, maxBinaryBytes)
	if err != nil {
		return err
	}
	b.Hashes[name] = contentHash(data)
	return os.WriteFile(filepath.Join(directory, name), data, 0700)
}
