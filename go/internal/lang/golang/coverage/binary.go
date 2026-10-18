package coverage

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// suite is one package's prebuilt coverage test binary, its top-level test
// names in list order, and the ones that call t.Parallel().
type suite struct {
	pkg      Package
	dir, bin string
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

// suite compiles pkg's test binary into a private directory with the same
// coverage flags as the module-wide run and lists its top-level tests.
func (r runner) suite(pkg Package) (suite, error) {
	parallel, err := parallelTests(pkg)
	if err != nil {
		return suite{}, err
	}
	dir, err := os.MkdirTemp(r.temp, "bin-")
	if err != nil {
		return suite{}, err
	}
	s := suite{pkg: pkg, dir: dir, bin: filepath.Join(dir, "pkg.test"), parallel: parallel}
	if _, err := r.command("test", "-c", "-covermode=atomic", "-coverpkg=./...", "-o", s.bin, pkg.ImportPath); err != nil {
		return suite{}, err
	}
	out, err := r.exec(s.bin, pkg.Dir, pkg.ImportPath+" -test.list", nil, "-test.list=^Test")
	s.names = testNames(string(out))
	return s, err
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
