package langs

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
	javaTests "github.com/may-journal/fitness-runner/go/internal/lang/java/tests"
	javaViews "github.com/may-journal/fitness-runner/go/internal/lang/java/views"
)

// Java is the Java language: a repo with a Maven or Gradle build file. A
// source in a test source set's folder is a test, and the architecture
// views read the rest.
var Java = lang.Module{
	Name:            "java",
	Detect:          java.Applies,
	Exts:            []string{java.Ext},
	Manifests:       java.Manifests(),
	Lockfiles:       []string{"*.lockfile"},
	BuildOutput:     java.BuildOutput,
	Wrappers:        java.Wrappers(),
	Tools:           []string{"java", java.Maven.Wrapper, java.Maven.Command, java.Gradle.Wrapper, java.Gradle.Command},
	Checks:          []string{"spotless", "checkstyle", "java-version", "java-test"},
	Technology:      java.Technology,
	TechnologyWords: strings.Fields("java jvm"),
	Runtime:         java.Runtime,
	Views:           javaViews.Reader,
	Inventory:       []string{java.Ext},
	Tests:           javaTests.Requirements,
}
