package coverage

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// suite is one package's prebuilt coverage test binary and its top-level
// test names, in the order the binary lists them.
type suite struct {
	pkg   Package
	bin   string
	names []string
}

// testedPackages keeps the packages that have test files, in order.
func testedPackages(packages []Package) []Package {
	var tested []Package
	for _, pkg := range packages {
		if len(pkg.TestGoFiles)+len(pkg.XTestGoFiles) > 0 {
			tested = append(tested, pkg)
		}
	}
	return tested
}

// auditPackage builds pkg's test binary once, audits it, and removes the
// binary before returning.
func (r runner) auditPackage(pkg Package) ([]testGroup, error) {
	dir, err := os.MkdirTemp(r.temp, "bin-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	s, err := r.suite(pkg, filepath.Join(dir, "pkg.test"))
	if err != nil {
		return nil, err
	}
	return r.auditSuite(s)
}

// suite compiles pkg's test binary to bin with the same coverage flags as
// the module-wide run and lists its top-level tests.
func (r runner) suite(pkg Package, bin string) (suite, error) {
	if _, err := r.command("test", "-c", "-covermode=atomic", "-coverpkg=./...", "-o", bin, pkg.ImportPath); err != nil {
		return suite{}, err
	}
	out, err := r.exec(bin, pkg.Dir, pkg.ImportPath+" -test.list", nil, "-test.list=^Test")
	if err != nil {
		return suite{}, err
	}
	return suite{pkg: pkg, bin: bin, names: testNames(string(out))}, nil
}

// testNames keeps the top-level test names from -test.list output.
func testNames(out string) []string {
	var names []string
	for _, name := range strings.Split(out, "\n") {
		if testName.MatchString(name) {
			names = append(names, name)
		}
	}
	return names
}

var testName = regexp.MustCompile(`^Test[\pL\pN_]*$`)

// measureBinary runs the suite's binary once for pattern, from the package
// directory as go test would, with a fresh profile and child coverage dir.
func (r runner) measureBinary(s suite, pattern string) (Profile, error) {
	return r.collect(func(profile, child string) error {
		counters := filepath.Join(filepath.Dir(profile), "counters")
		if err := os.Mkdir(counters, 0700); err != nil {
			return err
		}
		args := []string{"-test.paniconexit0", "-test.timeout=10m0s", "-test.count=1", "-test.run=" + pattern, "-test.coverprofile=" + profile, "-test.gocoverdir=" + counters}
		_, err := r.exec(s.bin, s.pkg.Dir, s.pkg.ImportPath+" test coverage", []string{"FITNESS_GO_COVER_DIR=" + child}, args...)
		return err
	})
}
