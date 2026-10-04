package release

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func archiveDirectory(path, directory string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	compressed := gzip.NewWriter(file)
	writer := tar.NewWriter(compressed)
	err = writeDirectory(writer, directory)
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	if closeErr := compressed.Close(); err == nil {
		err = closeErr
	}
	return err
}

func writeDirectory(writer *tar.Writer, directory string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := writeBinary(writer, directory, entry); err != nil {
			return err
		}
	}
	return nil
}

func writeBinary(writer *tar.Writer, directory string, entry os.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("bundle file %s is not regular", entry.Name())
	}
	header := &tar.Header{Name: filepath.Base(directory) + "/" + entry.Name(), Mode: 0755, Size: info.Size()}
	if err := writer.WriteHeader(header); err != nil {
		return err
	}
	return copyBinary(writer, filepath.Join(directory, entry.Name()))
}

func copyBinary(writer io.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	return err
}
