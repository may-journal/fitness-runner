package gocoverage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteAudit exports only on explicit request, outside the repository, to a new file.
func (r Report) WriteAudit(root, name string) error {
	if name == "" {
		return nil
	}
	file, err := auditDestination(root, name)
	if err != nil {
		return err
	}
	output, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	data := AuditMap{Version: 1, Basis: "Go statement blocks projected onto source lines; subtests grouped with parent; shared setup separate; unstable groups excluded", Modules: append([]ModuleMap{}, r.Maps...)}
	return encodeAudit(output, data)
}
func encodeAudit(file *os.File, data AuditMap) error {
	err := json.NewEncoder(file).Encode(data)
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(file.Name())
		return err
	}
	if closeErr != nil {
		_ = os.Remove(file.Name())
	}
	return closeErr
}
func auditDestination(root, name string) (string, error) {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return "", err
	}
	repo, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	if err := outsideRepo(repo, parent); err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(absolute)), nil
}
func outsideRepo(root, parent string) error {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absolute, parent)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil
	}
	return fmt.Errorf("write the audit map outside the repository")
}
