package lang

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// TestScanEnv names the file in which the runner shares one scan of the
// repo's requirement tests with every check that reads them.
const TestScanEnv = "FITNESS_TEST_SCAN"

// Scanned is one scan of every testing module's tests.
type Scanned struct {
	// Root is the repo the scan read.
	Root string
	// Modules names the testing modules the scan asked, in registry order.
	Modules []string
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
// the runner's shared scan instead when that scan asked the same modules of root.
func ScanTests(root string, modules []Module) (Scanned, error) {
	if s, ok := sharedScan(root, modules); ok {
		return s, nil
	}
	return scan(root, modules)
}

// scan reads every testing module's tests in root.
func scan(root string, modules []Module) (Scanned, error) {
	s := Scanned{Root: filepath.Clean(root), Modules: testingNames(modules)}
	for _, m := range Testing(modules) {
		if err := s.add(root, m); err != nil {
			return s, err
		}
	}
	return s, nil
}

// testingNames names the testing modules among modules, in registry order.
func testingNames(modules []Module) []string {
	var names []string
	for _, m := range Testing(modules) {
		names = append(names, m.Name)
	}
	return names
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

// ShareTestScan scans root afresh and writes the scan to a temporary file for
// the run's checks. Its env always replaces an inherited scan, and a scan that fails or outlasts wait shares none.
func ShareTestScan(root string, modules []Module, wait time.Duration) (env []string, remove func()) {
	none := []string{TestScanEnv + "="}
	s, ok := scanWithin(root, modules, wait)
	if !ok {
		return none, func() {}
	}
	name, err := writeScan(s)
	if err != nil {
		return none, func() {}
	}
	return []string{TestScanEnv + "=" + name}, func() { _ = os.Remove(name) }
}

// scanWithin scans root, giving up once wait passes; a scan given up on
// finishes unread.
func scanWithin(root string, modules []Module, wait time.Duration) (Scanned, bool) {
	type result struct {
		s   Scanned
		err error
	}
	done := make(chan result, 1)
	go func() {
		s, err := scan(root, modules)
		done <- result{s, err}
	}()
	select {
	case r := <-done:
		return r.s, r.err == nil
	case <-time.After(wait):
		return Scanned{}, false
	}
}

// writeScan writes s to a new temporary file and returns its name.
func writeScan(s Scanned) (string, error) {
	data, _ := json.Marshal(s)
	f, err := os.CreateTemp("", "fitness-tests-*.json")
	if err != nil {
		return "", err
	}
	_, err = f.Write(data)
	if err = errors.Join(err, f.Close()); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// sharedScan reads the scan the runner shared, when it asked the same
// modules of root.
func sharedScan(root string, modules []Module) (Scanned, bool) {
	var s Scanned
	name := os.Getenv(TestScanEnv)
	if name == "" {
		return s, false
	}
	data, err := os.ReadFile(name)
	if err != nil || json.Unmarshal(data, &s) != nil {
		return s, false
	}
	return s, s.Root == filepath.Clean(root) && slices.Equal(s.Modules, testingNames(modules))
}
