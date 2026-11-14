package java

// cspell:ignore temurin

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// NoJDK is the failure a check reports when no JDK runs on the machine.
const NoJDK = "Java not installed: install a JDK, such as Temurin from https://adoptium.net, or set JAVA_HOME"

// versionRe reads the quoted version from `java -version`, such as
// openjdk version "21.0.2" or java version "1.8.0_392".
var versionRe = regexp.MustCompile(`version "([^"]+)"`)

// JDK reports the running JDK's version, as JAVA_HOME's java or else the
// java on PATH prints it, and whether one runs at all. A launcher stub that
// exits with an error, as macOS ships without a JDK, counts as none.
func JDK() (version string, ok bool) {
	bin, ok := javaBin()
	if !ok {
		return "", false
	}
	out, err := exec.Command(bin, "-version").CombinedOutput()
	if err != nil {
		return "", false
	}
	m := versionRe.FindStringSubmatch(string(out))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// javaBin finds the java launcher: JAVA_HOME's, else the one on PATH.
func javaBin() (string, bool) {
	if home := os.Getenv("JAVA_HOME"); home != "" && executable(filepath.Join(home, "bin", "java")) {
		return filepath.Join(home, "bin", "java"), true
	}
	bin, err := exec.LookPath("java")
	return bin, err == nil
}

// majorRe reads the leading number run of a version, such as 17.0.9 in
// temurin-17.0.9+9.
var majorRe = regexp.MustCompile(`\d+(?:\.\d+)?`)

// Major reads a Java version's feature release: 21 from 21.0.2 or
// temurin-21, and 8 from the old 1.8 form.
func Major(version string) (int, bool) {
	m := majorRe.FindString(version)
	if m == "" {
		return 0, false
	}
	first, rest, _ := strings.Cut(m, ".")
	if first == "1" && rest != "" {
		first = rest
	}
	n, err := strconv.Atoi(first)
	return n, err == nil
}
