package coverage

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// suite is one package's prebuilt coverage test binary, its top-level test
// names in list order, and which may run in parallel.
type suite struct {
	pkg      Package
	bin      string
	names    []string
	parallel map[string]bool
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

// suite compiles pkg's test binary to bin with the module-wide run's coverage
// flags and lists its top-level tests, holding a process slot.
func (r runner) suite(pkg Package, bin string, parallel map[string]bool) (suite, error) {
	release, err := r.slot(pkg.ImportPath + " test build")
	if err != nil {
		return suite{}, err
	}
	defer release()
	if _, err := r.command("test", "-c", "-covermode=atomic", "-coverpkg=./...", "-o", bin, pkg.ImportPath); err != nil {
		return suite{}, err
	}
	out, err := r.exec(bin, pkg.Dir, pkg.ImportPath+" -test.list", nil, "-test.list=^Test")
	return suite{pkg: pkg, bin: bin, names: testNames(string(out)), parallel: parallel}, err
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

// measureBinary runs the suite's binary once for pattern in a process slot,
// from the package directory, with a fresh profile.
func (r runner) measureBinary(s suite, pattern string) (Profile, error) {
	release, err := r.slot(s.pkg.ImportPath + " test coverage")
	if err != nil {
		return nil, err
	}
	defer release()
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
