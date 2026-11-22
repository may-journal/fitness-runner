package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/actions"
)

func init() {
	register("build-release-bundle", "Build the release bundles and tools with GoReleaser and assemble the release, as a snapshot when SNAPSHOT is true.", nil, buildReleaseBundle)
	register("inspect-publication", "Write whether the release for RELEASE_TAG is still a draft, failing on any other answer.", []string{"draft"}, inspectPublication)
	register("write-release-notes", "Write the release notes from the changelog entry files onto the draft release, with the Linux release tool.", nil, releaseToolStep("release-notes"))
	register("promote-release", "Promote the verified release with the Linux release tool, using the automation App's token in GH_TOKEN.", nil, releaseToolStep("promote"))
	register("publish-go-module-tag", "Tag the release commit as the Go module version, failing when an existing tag names another commit.", nil, publishGoModuleTag)
	register("move-engine-major-tag", "Move the major version tag, such as v1, to RELEASE_TAG when that release is the newest.", nil, moveEngineMajorTag)
	register("run-verifier", "Run one command of the Go release verifier in VERIFIER, such as smoke or verify-download.", nil, runVerifier)
	register("record-binary-sizes", "Record binary sizes and install time for PLATFORM into the performance folder, using the VERIFIER program.", nil, recordBinarySizes)
	register("run-smoke-suite", "Run the full suite, with the pinned checks, through the candidate Go installer for PLATFORM.", nil, runSmokeSuite)
	register("run-requirement-shard", "Run shard SHARD of two of the requirement tests with the candidate Go installer for PLATFORM.", nil, runRequirementShard)
}

// linuxReleaseTool is the release tool the release jobs run on Linux.
const linuxReleaseTool = "out/fitness-release-linux-amd64"

// releaseTool runs fitness-release from go/, as a maintainer does, and
// returns what it printed.
func releaseTool(args ...string) (string, error) {
	return actions.Call{}.Capture("go", append([]string{"-C", "go", "run", "./cmd/fitness-release"}, append(args, "--root", "..")...)...)
}

// goreleaser runs one GoReleaser pass with config and extra environment.
func goreleaser(config string, extra ...string) error {
	args := []string{"release", "--clean", "--skip=publish"}
	if env("SNAPSHOT") == "true" {
		args = append(args, "--snapshot")
	}
	return actions.Call{Env: extra}.Run("goreleaser", append(args, "--config", config)...)
}

func buildReleaseBundle(actions.Step, []string) error {
	version, err := releaseTool("version")
	if err != nil {
		return err
	}
	if err := goreleaser(".goreleaser-bundles.yml", "FITNESS_VERSION="+version); err != nil {
		return err
	}
	hashes, err := releaseTool("bundle-hashes")
	if err != nil {
		return err
	}
	return firstError(
		func() error {
			return goreleaser(".goreleaser-tools.yml", "FITNESS_VERSION="+version, "FITNESS_BUNDLE_HASHES="+hashes)
		},
		func() error {
			return actions.Run("go", "-C", "go", "run", "./cmd/fitness-release", "assemble", "--root", "..")
		},
	)
}

func inspectPublication(step actions.Step, _ []string) error {
	draft, err := actions.Gh("release", "view", env("RELEASE_TAG"), "--json", "isDraft", "--jq", ".isDraft")
	if err != nil {
		return err
	}
	if draft != "true" && draft != "false" {
		return fmt.Errorf("gh answered %q for whether the release is a draft", draft)
	}
	return step.Output("draft", draft)
}

// releaseToolStep runs one command of the Linux release tool the bundle job
// built, so fitness-release keeps owning the release logic.
func releaseToolStep(command string) func(actions.Step, []string) error {
	return func(actions.Step, []string) error {
		return actions.RunTool(linuxReleaseTool, command)
	}
}

func publishGoModuleTag(actions.Step, []string) error {
	tag := env("RELEASE_TAG")
	release, err := actions.Git("rev-list", "-n", "1", tag)
	if err != nil {
		return err
	}
	module := "go/" + tag
	if _, _, err := (actions.Call{}).Quiet("git", "rev-parse", "--verify", module); err == nil {
		return checkModuleTag(module, release)
	}
	return firstError(
		func() error { return actions.Run("git", "tag", module, release) },
		func() error { return actions.Run("git", "push", "origin", "refs/tags/"+module) },
	)
}

// checkModuleTag fails unless the existing module tag names release.
func checkModuleTag(module, release string) error {
	tagged, err := actions.Git("rev-list", "-n", "1", module)
	if err != nil {
		return err
	}
	if tagged != release {
		return actions.Fail(fmt.Sprintf("Tag %s names %s, not the release commit %s.", module, tagged, release))
	}
	return nil
}

func moveEngineMajorTag(actions.Step, []string) error {
	tag := env("RELEASE_TAG")
	latest, err := actions.Gh("release", "view", "--json", "tagName", "--jq", ".tagName")
	if err != nil || latest != tag {
		return err
	}
	major, _, _ := strings.Cut(tag, ".")
	commit, err := actions.Git("rev-list", "-n", "1", tag)
	if err != nil {
		return err
	}
	return firstError(
		func() error { return actions.Run("git", "tag", "-f", major, commit) },
		func() error { return actions.Run("git", "push", "-f", "origin", "refs/tags/"+major) },
	)
}

func runVerifier(_ actions.Step, args []string) error {
	if len(args) != 1 {
		return errors.New("name one verifier command, such as smoke or verify-download")
	}
	return actions.RunTool(env("VERIFIER"), args[0])
}

func recordBinarySizes(actions.Step, []string) error {
	dir := filepath.Join(env("RUNNER_TEMP"), "fitness-performance")
	verifier := env("VERIFIER")
	if err := firstError(func() error { return os.MkdirAll(dir, 0o755) }, func() error { return actions.MakeExecutable(verifier) }); err != nil {
		return err
	}
	record, err := actions.Call{}.Capture(verifier, "record")
	if err != nil {
		return err
	}
	// Only a finished record reaches the folder the artifact keeps.
	name := "release-" + env("PLATFORM") + ".json"
	part := filepath.Join(env("RUNNER_TEMP"), name)
	if err := os.WriteFile(part, []byte(record+"\n"), 0o644); err != nil {
		return err
	}
	return os.Rename(part, filepath.Join(dir, name))
}

// candidateInstaller is the one candidate Go installer for PLATFORM in out/,
// made executable.
func candidateInstaller() (string, error) {
	found, err := filepath.Glob("out/fitness-install-*-" + env("PLATFORM"))
	if err == nil && len(found) != 1 {
		err = fmt.Errorf("want one candidate installer for %s in out, found %d", env("PLATFORM"), len(found))
	}
	if err != nil {
		return "", err
	}
	return found[0], actions.MakeExecutable(found[0])
}

func runSmokeSuite(actions.Step, []string) error {
	installer, err := candidateInstaller()
	if err != nil {
		return err
	}
	return actions.Run("./"+installer, "--with-checks", "--", "--policy=external")
}

func runRequirementShard(actions.Step, []string) error {
	installer, err := candidateInstaller()
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(installer)
	if err != nil {
		return err
	}
	return actions.Call{Env: []string{"FITNESS_INSTALL=" + abs}}.Run("sh", "tests/requirements-shard.sh", env("SHARD"), "2")
}
