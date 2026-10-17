package langs

import (
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

// Java is the Java language: a repo with a Maven or Gradle build file. Its
// tests and architecture views are not read yet, so its sources stay in the
// sources folder.
var Java = lang.Module{
	Name:        "java",
	Detect:      java.Applies,
	Manifests:   java.Manifests(),
	Lockfiles:   []string{"*.lockfile"},
	BuildOutput: java.BuildOutput,
	Tools:       []string{"java", java.Maven.Wrapper, java.Maven.Command, java.Gradle.Wrapper, java.Gradle.Command},
	Checks:      []string{"spotless", "checkstyle", "java-version"},
	Dicts:       []string{"java.txt"},
}
