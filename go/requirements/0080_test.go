// cspell:ignore jdks foojay adoptium hotspot openj
package requirements

import (
	"path/filepath"
	"strings"
	"testing"
)

// javaVersionRun runs `fitness-install -- java-version` on a Java repo with
// files written over it and a JDK 21 running.
func javaVersionRun(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	_, out, code := javaRun(t, javaRepo(t, files), "java-version", nil)
	return out, code
}

// javaVersionRelease is a Maven build whose compiler release is release.
func javaVersionRelease(release string) string {
	return "<project><properties><maven.compiler.release>" + release + "</maven.compiler.release></properties></project>\n"
}

func Test0080_1(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{"pom.xml": "<project/>\n"})
	sees(t, out, code, 1, "no JDK version is pinned; add a .java-version file naming the release",
		"JavaLanguageVersion.of(21)")
}

func Test0080_2(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{"pom.xml": "<project/>\n", ".java-version": "17\n"})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0080_3(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{"pom.xml": javaVersionRelease("25")})
	sees(t, out, code, 1, "JDK 21.0.2 does not satisfy pom.xml (requires 25); install JDK 25 or point JAVA_HOME at one")
}

func Test0080_4(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{
		"pom.xml": "<project><properties><java.version>25</java.version>" +
			"<maven.compiler.release>${java.version}</maven.compiler.release></properties></project>\n",
	})
	sees(t, out, code, 1, "does not satisfy pom.xml (requires 25)")
	// A parent from the local Maven repository maps the pom's property to
	// the compiler release, as framework parents do.
	home := t.TempDir()
	write(t, home, map[string]string{".m2/repository/" + javaVersionParentPom: javaVersionParent})
	child := "<project><parent><groupId>com.example</groupId><artifactId>parent</artifactId>" +
		"<version>1</version><relativePath/></parent><properties><java.version>25</java.version></properties></project>\n"
	javaVersionSees(t, javaRepo(t, map[string]string{"pom.xml": child}), home, "JDK 21.0.2 does not satisfy pom.xml (requires 25)")
	// The user's Maven settings may move the local repository.
	moved := t.TempDir()
	write(t, moved, map[string]string{".m2/settings.xml": "<settings><localRepository>${user.home}/maven</localRepository></settings>\n", "maven/" + javaVersionParentPom: javaVersionParent})
	javaVersionSees(t, javaRepo(t, map[string]string{"pom.xml": child}), moved, "does not satisfy pom.xml (requires 25)")
	// A pom outside the repo at the default relative path is no parent in
	// the repo, so the local repository's parent still counts.
	repo := javaRepo(t, map[string]string{"pom.xml": strings.Replace(child, "<relativePath/>", "", 1)})
	write(t, filepath.Dir(repo), map[string]string{"pom.xml": "<project><artifactId>parent</artifactId></project>\n"})
	javaVersionSees(t, repo, home, "does not satisfy pom.xml (requires 25)")
	// A module takes the property its parent in the repo sets.
	javaVersionSees(t, javaRepo(t, map[string]string{
		"pom.xml": "<project><artifactId>app</artifactId><modules><module>core</module></modules><properties><java.version>25</java.version></properties></project>\n",
		"core/pom.xml": "<project><parent><groupId>com.example</groupId><artifactId>app</artifactId><version>1</version></parent>" +
			"<properties><maven.compiler.release>${java.version}</maven.compiler.release></properties></project>\n",
	}), home, "does not satisfy core/pom.xml (requires 25)")
}

// javaVersionParentPom is where a local Maven repository keeps the parent
// pom com.example:parent:1.
const javaVersionParentPom = "com/example/parent/1/parent-1.pom"

// javaVersionParent maps java.version to the compiler release, as
// framework parents do.
const javaVersionParent = "<project><properties><java.version>17</java.version>" +
	"<maven.compiler.release>${java.version}</maven.compiler.release></properties></project>\n"

// javaVersionSees runs java-version on repo with home as the user's home
// and asserts it fails with want.
func javaVersionSees(t *testing.T, repo, home, want string) {
	t.Helper()
	rp := standIn(t, map[string][]response{"java": {javaJDK}})
	out, code := fitness(t, repo, append(rp.env, "JAVA_HOME=", "HOME="+home), "java-version")
	sees(t, out, code, 1, want)
}

func Test0080_5(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{"pom.xml": "<project/>\n", ".java-version": "system\n"})
	sees(t, out, code, 1, `.java-version pins JDK "system", which names no release`)
}

func Test0080_6(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{"pom.xml": javaVersionRelease("21")})
	out, code := fitness(t, repo, []string{pathWithout(t, "java"), "JAVA_HOME="}, "java-version")
	sees(t, out, code, 1, "Java not installed: install a JDK")
}

// javaVersionNoGradle is a Gradle stand-in that fails if it ever runs, since
// the check must find toolchain JDKs without starting Gradle.
var javaVersionNoGradle = response{Stderr: "gradle must not run\n", Exit: 1}

// javaVersionMissing30 is the failure for a toolchain release no installed
// JDK has. No machine holds a JDK 30, so fixture folders decide the result.
const javaVersionMissing30 = "build.gradle asks for a JDK 30 toolchain, which is not installed; install JDK 30"

// javaVersionToolchain is a Gradle build whose toolchain asks for release.
func javaVersionToolchain(t *testing.T, release string, extra map[string]string) string {
	t.Helper()
	return javaVersionBuild(t, "languageVersion = JavaLanguageVersion.of("+release+")", extra)
}

// javaVersionBuild is a Gradle build whose toolchain block holds body.
func javaVersionBuild(t *testing.T, body string, extra map[string]string) string {
	t.Helper()
	files := map[string]string{"build.gradle": "java { toolchain { " + body + " } }\n"}
	for name, text := range extra {
		files[name] = text
	}
	return javaRepo(t, files)
}

// javaVersionJDK is the release file of an installed JDK of release.
func javaVersionJDK(release string) string {
	return javaVersionVendorJDK(release, "Eclipse Adoptium", "Hotspot")
}

// javaVersionVendorJDK is the release file of a JDK of release from vendor
// running the JVM variant.
func javaVersionVendorJDK(release, vendor, variant string) string {
	return "JAVA_VERSION=\"" + release + ".0.1\"\nIMPLEMENTOR=\"" + vendor + "\"\nJVM_VARIANT=\"" + variant + "\"\n"
}

// javaVersionGradleRun runs java-version on repo with home as the user's
// home and a Gradle stand-in, failing the test if Gradle ran.
func javaVersionGradleRun(t *testing.T, repo, home string) (string, int) {
	t.Helper()
	rp := standIn(t, map[string][]response{"java": {javaJDK}, "gradle": {javaVersionNoGradle}})
	out, code := fitness(t, repo, append(rp.env, "JAVA_HOME=", "GRADLE_USER_HOME=", "HOME="+home), "java-version")
	if calls := rp.calls("gradle"); len(calls) != 0 {
		t.Errorf("gradle calls = %q, want none", calls)
	}
	return out, code
}

// javaVersionPasses asserts java-version passes repo with home.
func javaVersionPasses(t *testing.T, repo, home string) {
	t.Helper()
	out, code := javaVersionGradleRun(t, repo, home)
	sees(t, out, code, 0, "All 1 checks passed")
}

// javaVersionFails asserts java-version fails repo with home, naming a
// missing JDK 30.
func javaVersionFails(t *testing.T, repo, home string) {
	t.Helper()
	out, code := javaVersionGradleRun(t, repo, home)
	sees(t, out, code, 1, javaVersionMissing30)
}

// javaVersionHome is a user home holding the files.
func javaVersionHome(t *testing.T, files map[string]string) string {
	t.Helper()
	home := t.TempDir()
	write(t, home, files)
	return home
}

// javaVersionFolder is a folder holding a JDK 30 release file.
func javaVersionFolder(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, map[string]string{"release": javaVersionJDK("30")})
	return dir
}

// javaVersionResolver is a settings script that registers a download
// repository.
const javaVersionResolver = "toolchainManagement { jvm { javaRepositories { repository('foojay') { resolverClass = FoojayResolver } } } }\n"

func Test0080_8(t *testing.T) {
	t.Parallel()
	work := func(release string, extra map[string]string) string { return javaVersionToolchain(t, release, extra) }
	// No installed JDK has the release and no resolver is configured. Gradle
	// never runs, so the check stays fast.
	javaVersionFails(t, work("30", nil), t.TempDir())
	// Gradle's own download folder counts, flat or as a macOS bundle one
	// folder down.
	javaVersionPasses(t, work("30", nil), javaVersionHome(t, map[string]string{".gradle/jdks/jdk-30/release": javaVersionJDK("30")}))
	javaVersionPasses(t, work("30", nil), javaVersionHome(t, map[string]string{".gradle/jdks/amazon-30/jdk-30/Contents/Home/release": javaVersionJDK("30")}))
	// Gradle always finds the JDK it runs on.
	javaVersionPasses(t, work("21", nil), t.TempDir())
	// An empty JAVA_HOME, home or environment name reads nothing from the
	// working folder, where a release file sits.
	cwd := work("30", map[string]string{"release": javaVersionJDK("30"), "gradle.properties": "org.gradle.java.installations.fromEnv=NO_SUCH_JDK_VARIABLE\n"})
	javaVersionFails(t, cwd, "")
	javaVersionPropertiesCases(t)
	javaVersionResolverCases(t)
	javaVersionVendorCases(t)
}

// javaVersionPropertiesCases covers the gradle.properties forms, build root
// and precedence.
func javaVersionPropertiesCases(t *testing.T) {
	jdks := javaVersionFolder(t)
	paths := func(text string) map[string]string { return map[string]string{"gradle.properties": text} }
	// gradle.properties lists JDK folders in every properties form: an equals
	// sign, a colon or a space, and a list continued across lines.
	for _, text := range []string{
		"org.gradle.java.installations.paths=" + jdks + "\n",
		"org.gradle.java.installations.paths : " + jdks + "\n",
		"org.gradle.java.installations.paths " + jdks + "\n",
		"# the list\norg.gradle.java.installations.paths=/no/such,\\\n    " + jdks + "\n",
	} {
		javaVersionPasses(t, javaVersionToolchain(t, "30", paths(text)), t.TempDir())
	}
	// The list splits on commas only, so a folder name may hold a colon.
	colon := filepath.Join(t.TempDir(), "a:b")
	write(t, colon, map[string]string{"release": javaVersionJDK("30")})
	javaVersionPasses(t, javaVersionToolchain(t, "30", paths("org.gradle.java.installations.paths="+colon+"\n")), t.TempDir())
	// A build in a folder reads its own gradle.properties, not the repo's.
	nested := javaRepo(t, map[string]string{
		"app/build.gradle":  "java { toolchain { languageVersion = JavaLanguageVersion.of(30) } }\n",
		"gradle.properties": "org.gradle.java.installations.paths=" + jdks + "\n",
	})
	out, code := javaVersionGradleRun(t, nested, t.TempDir())
	sees(t, out, code, 1, "app/build.gradle asks for a JDK 30 toolchain")
	// The Gradle user home's gradle.properties overrides the build's.
	repo := javaVersionToolchain(t, "30", paths("org.gradle.java.installations.paths="+jdks+"\n"))
	javaVersionFails(t, repo, javaVersionHome(t, map[string]string{".gradle/gradle.properties": "org.gradle.java.installations.paths=\n"}))
}

// javaVersionResolverCases covers what counts as a toolchain resolver.
func javaVersionResolverCases(t *testing.T) {
	settings := func(text string) map[string]string { return map[string]string{"settings.gradle": text} }
	build := func(extra map[string]string) string { return javaVersionToolchain(t, "30", extra) }
	// A registered download repository, or the Foojay plugin, lets Gradle
	// download the JDK.
	javaVersionPasses(t, build(settings(javaVersionResolver)), t.TempDir())
	javaVersionPasses(t, build(settings("plugins { id 'org.gradle.toolchains.foojay-resolver-convention' version '1.0.0' }\n")), t.TempDir())
	// So does one in an init script of the Gradle user home.
	javaVersionPasses(t, build(nil), javaVersionHome(t, map[string]string{".gradle/init.d/resolver.gradle": javaVersionResolver}))
	// An empty block registers none, and a commented-out one is no block.
	javaVersionFails(t, build(settings("toolchainManagement { jvm { } }\n")), t.TempDir())
	javaVersionFails(t, build(settings("// id 'org.gradle.toolchains.foojay-resolver-convention'\n/* "+javaVersionResolver+" */\n")), t.TempDir())
	// Turning auto-download off wins over a resolver.
	off := settings(javaVersionResolver)
	off["gradle.properties"] = "org.gradle.java.installations.auto-download=false\n"
	javaVersionFails(t, build(off), t.TempDir())
}

// javaVersionVendorCases covers vendor and implementation constraints.
func javaVersionVendorCases(t *testing.T) {
	home := func(vendor, variant string) string {
		return javaVersionHome(t, map[string]string{".gradle/jdks/jdk-30/release": javaVersionVendorJDK("30", vendor, variant)})
	}
	adoptium := func() string {
		return javaVersionBuild(t, "languageVersion = JavaLanguageVersion.of(30); vendor = JvmVendorSpec.ADOPTIUM", nil)
	}
	// A toolchain that names a vendor needs a JDK from it.
	javaVersionPasses(t, adoptium(), home("Eclipse Adoptium", "Hotspot"))
	javaVersionFails(t, adoptium(), home("Amazon.com Inc.", "Hotspot"))
	// So does one that asks for the J9 implementation.
	j9 := func() string {
		return javaVersionBuild(t, "languageVersion = JavaLanguageVersion.of(30); implementation = JvmImplementation.J9", nil)
	}
	javaVersionPasses(t, j9(), home("IBM Corporation", "Openj9"))
	javaVersionFails(t, j9(), home("IBM Corporation", "Hotspot"))
}

func Test0080_7(t *testing.T) {
	t.Parallel()
	_, out, code := javaRun(t, example(t, "happyRepo", nil), "java-version", nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "java-version")
}
