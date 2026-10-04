package release

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const externalChecks = "markdown-filename-kebab-case,markdown-links"
const externalPolicy = "--policy=external"
const checksFlag = "--checks=" + externalChecks

func (s smoke) command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = s.repo
	cmd.Env = s.env
	cmd.Stderr = os.Stderr
	return cmd
}

func (s smoke) run(ctx context.Context, name string, args ...string) error {
	cmd := s.command(ctx, name, args...)
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func (s smoke) check(ctx context.Context) error {
	steps := []func(context.Context) error{
		s.prepareRepo,
		func(ctx context.Context) error {
			return s.run(ctx, s.installer, "--", externalPolicy, checksFlag, "--all")
		},
		s.checkHookTools, s.checkAction, s.installHook, s.rejectInvalid,
	}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s smoke) prepareRepo(ctx context.Context) error {
	if err := s.run(ctx, "git", "init", "-q"); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.repo, "readme.md"), []byte("# Example\n"), 0600); err != nil {
		return err
	}
	return s.prepareExternalConfig(ctx)
}

func (s smoke) checkAction(ctx context.Context) error {
	cmd := s.command(ctx, s.installer, "--github-action")
	cmd.Env = append(cmd.Env, "FITNESS_INSTALL_ONLY=true", "FITNESS_POLICY=external", "FITNESS_CHECKS="+externalChecks, "GITHUB_PATH="+filepath.Join(s.work, "path"), "GITHUB_OUTPUT="+filepath.Join(s.work, "outputs"))
	if err := cmd.Run(); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(s.work, "outputs"))
	if err != nil {
		return err
	}
	bin := strings.TrimSpace(strings.TrimPrefix(string(data), "bin="))
	return s.run(ctx, filepath.Join(bin, runnerName), "--help")
}

func (s smoke) installHook(ctx context.Context) error {
	if err := s.run(ctx, s.installer, "--install-hook", "--", externalPolicy, checksFlag, "--all"); err != nil {
		return err
	}
	return s.commit(ctx, "valid")
}

func (s smoke) commit(ctx context.Context, message string) error {
	return s.run(ctx, "git", "-c", "user.name=Fitness", "-c", "user.email=fitness@example.com", "commit", "-qm", message)
}

func (s smoke) rejectInvalid(ctx context.Context) error {
	if err := os.WriteFile(filepath.Join(s.repo, "Bad_Name.md"), []byte("# Example\n"), 0600); err != nil {
		return err
	}
	if err := s.run(ctx, "git", "add", "Bad_Name.md"); err != nil {
		return err
	}
	if err := s.expectFailure(ctx); err != nil {
		return err
	}
	if err := s.commit(ctx, "invalid"); err == nil {
		return fmt.Errorf("hook allowed an invalid commit")
	}
	return nil
}

func (s smoke) expectFailure(ctx context.Context) error {
	err := s.run(ctx, s.installer, "--", externalPolicy, checksFlag, "--all")
	status, ok := err.(*exec.ExitError)
	if !ok {
		return fmt.Errorf("expected a failing check, got %v", err)
	}
	if status.ExitCode() != 1 {
		return fmt.Errorf("check exit status changed: %d", status.ExitCode())
	}
	return nil
}

func (s smoke) prepareExternalConfig(ctx context.Context) error {
	config := externalConfig
	if err := os.WriteFile(filepath.Join(s.repo, ".fitnessrc.json"), []byte(config), 0600); err != nil {
		return err
	}
	if err := s.run(ctx, "git", "add", "readme.md", ".fitnessrc.json"); err != nil {
		return err
	}
	return s.checkDefaultSuite(ctx)
}

// The fixture lacks a valid commit and changelog, so all-check execution fails.
// Verify every shipped check runs; none may be hidden by external policy.
func (s smoke) checkDefaultSuite(ctx context.Context) error {
	output, err := s.command(ctx, s.installer, "--install-only").Output()
	if err != nil {
		return err
	}
	names, err := filepath.Glob(filepath.Join(strings.TrimSpace(string(output)), "fitness-check-*"))
	if err != nil {
		return err
	}
	command := s.command(ctx, s.installer, "--", externalPolicy, "--all")
	command.Stderr = nil
	output, err = command.CombinedOutput()
	status, ok := err.(*exec.ExitError)
	if !ok || status.ExitCode() != 1 {
		return fmt.Errorf("full-suite fixture must fail checks: %v\n%s", err, output)
	}
	return requireAllChecks(names, string(output))
}

func requireAllChecks(paths []string, output string) error {
	if len(paths) == 0 {
		return fmt.Errorf("bundle contains no check binaries")
	}
	for _, path := range paths {
		name := strings.TrimPrefix(filepath.Base(path), "fitness-check-")
		// Coverage remains explicitly selected during its approved rollout.
		if name == "go-test-coverage" {
			continue
		}
		if !strings.Contains(output, "→ "+name+"\n") {
			return fmt.Errorf("external default omitted check %s", name)
		}
	}
	return nil
}
