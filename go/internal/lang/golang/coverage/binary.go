package coverage

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/par"
)

// suite is one package's prebuilt coverage test binary and its top-level
// test names, in the order the binary lists them.
type suite struct {
	pkg   Package
	bin   string
	names []string
}

// suites builds each tested package's binary once, on the shared process
// pool, and returns them in package order or the first error in that order.
func (r runner) suites(packages []Package) ([]suite, error) {
	var tested []Package
	for _, pkg := range packages {
		if len(pkg.TestGoFiles)+len(pkg.XTestGoFiles) > 0 {
			tested = append(tested, pkg)
		}
	}
	built := par.Map(len(tested), par.Processes(), func(i int) built {
		s, err := r.suite(tested[i])
		return built{s, err}
	})
	result := make([]suite, len(built))
	for i, b := range built {
		if b.err != nil {
			return nil, b.err
		}
		result[i] = b.suite
	}
	return result, nil
}

type built struct {
	suite suite
	err   error
}

// suite compiles pkg's test binary with the same coverage flags as the
// module-wide run and lists its top-level tests.
func (r runner) suite(pkg Package) (suite, error) {
	dir, err := os.MkdirTemp(r.temp, "bin-")
	if err != nil {
		return suite{}, err
	}
	bin := filepath.Join(dir, "pkg.test")
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

// measured is one run of a suite's binary: its profile and wall time.
type measured struct {
	profile  Profile
	duration time.Duration
	err      error
}

// measureBinary runs the suite's binary once for pattern, from the package
// directory as go test would, with a fresh profile and child coverage dir.
func (r runner) measureBinary(s suite, pattern string) measured {
	start := time.Now()
	p, err := r.collect(func(profile, child string) error {
		counters := filepath.Join(filepath.Dir(profile), "counters")
		if err := os.Mkdir(counters, 0700); err != nil {
			return err
		}
		args := []string{"-test.paniconexit0", "-test.timeout=10m0s", "-test.count=1", "-test.run=" + pattern, "-test.coverprofile=" + profile, "-test.gocoverdir=" + counters}
		_, err := r.exec(s.bin, s.pkg.Dir, s.pkg.ImportPath+" test coverage", []string{"FITNESS_GO_COVER_DIR=" + child}, args...)
		return err
	})
	return measured{profile: p, duration: time.Since(start), err: err}
}
