package lang

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// TestScanEnv names the file in which the runner shares one scan of the
// repo's requirement tests with every check that reads them.
const TestScanEnv = "FITNESS_TEST_SCAN"

// Scanned is one scan of every testing module's tests.
type Scanned struct {
	// Root is the repo the scan read.
	Root string
	// Tests are the tests found, in registry order.
	Tests []ModuleTest
	// Files are the files holding tests.
	Files []string
	// Written names the modules whose language the repo uses.
	Written []string
}

// ModuleTest is a requirement test and the name of the module that found it.
type ModuleTest struct {
	// Test is the test as its module found it.
	Test
	// Module names the module that found the test.
	Module string
}

// ScanTests reads every testing module's tests in registry order. It reads
// the runner's shared scan instead when that scan is of root.
func ScanTests(root string, modules []Module) (Scanned, error) {
	if s, ok := sharedScan(root); ok {
		return s, nil
	}
	s := Scanned{Root: filepath.Clean(root)}
	for _, m := range Testing(modules) {
		if err := s.add(root, m); err != nil {
			return s, err
		}
	}
	return s, nil
}

// add scans root with module m and keeps what it found.
func (s *Scanned) add(root string, m Module) error {
	tests, files, written, err := m.Tests.Scan(root)
	for _, t := range tests {
		s.Tests = append(s.Tests, ModuleTest{Test: t, Module: m.Name})
	}
	s.Files = append(s.Files, files...)
	if written {
		s.Written = append(s.Written, m.Name)
	}
	return err
}

// ShareTestScan scans root once and writes the scan to a temporary file for
// the run's checks. On failure each check scans for itself.
func ShareTestScan(root string, modules []Module) (env []string, remove func()) {
	none := func() {}
	s, err := ScanTests(root, modules)
	if err != nil {
		return nil, none
	}
	data, _ := json.Marshal(s)
	f, err := os.CreateTemp("", "fitness-tests-*.json")
	if err != nil {
		return nil, none
	}
	_, err = f.Write(data)
	if err = errors.Join(err, f.Close()); err != nil {
		_ = os.Remove(f.Name())
		return nil, none
	}
	return []string{TestScanEnv + "=" + f.Name()}, func() { _ = os.Remove(f.Name()) }
}

// sharedScan reads the scan the runner shared, when it is of root.
func sharedScan(root string) (Scanned, bool) {
	var s Scanned
	name := os.Getenv(TestScanEnv)
	if name == "" {
		return s, false
	}
	data, err := os.ReadFile(name)
	if err != nil || json.Unmarshal(data, &s) != nil {
		return s, false
	}
	return s, s.Root == filepath.Clean(root)
}
