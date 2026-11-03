// cspell:ignore mispeled
package requirements

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// misspelledNote is a note cspell fails.
const misspelledNote = "A mispeled wrd.\n"

// hookEvent pipes one Claude Code hook event, as JSON, into the installed
// `fitness claude-hook` run from repo, as the settings hooks run it. It
// asserts stdout holds nothing but the JSON Claude Code parses, and returns
// stdout and stderr and the exit code.
func hookEvent(t *testing.T, repo string, event map[string]any) (string, int) {
	t.Helper()
	event["cwd"] = repo
	raw, err := json.Marshal(event)
	mustDo(t, err)
	cmd := exec.Command(filepath.Join(installOnly(t), "fitness"), "claude-hook")
	cmd.Dir, cmd.Env, cmd.Stdin = repo, userEnv(), strings.NewReader(string(raw))
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run()
	onlyJSON(t, stdout.String())
	return stdout.String() + stderr.String(), cmd.ProcessState.ExitCode()
}

// onlyJSON asserts out is empty or one JSON object and nothing else.
func onlyJSON(t *testing.T, out string) {
	t.Helper()
	var verdict map[string]any
	if out != "" && json.Unmarshal([]byte(out), &verdict) != nil {
		t.Errorf("stdout must hold only the hook's JSON:\n%s", out)
	}
}

// editEvent is the PostToolUse event for a Write of file in repo.
func editEvent(repo, file string) map[string]any {
	return map[string]any{"hook_event_name": "PostToolUse", "tool_name": "Write", "tool_input": map[string]any{"file_path": filepath.Join(repo, file)}}
}

// bashEvent is the PreToolUse event for a Bash call running command.
func bashEvent(command string) map[string]any {
	return map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Bash", "tool_input": map[string]any{"command": command}}
}

// stopEvent is the Stop event, with stop_hook_active as active.
func stopEvent(active bool) map[string]any {
	return map[string]any{"hook_event_name": "Stop", "stop_hook_active": active}
}

// silentPass asserts the hook let the event pass without a word.
func silentPass(t *testing.T, out string, code int) {
	t.Helper()
	if code != 0 || out != "" {
		t.Errorf("want a silent pass, got exit %d:\n%s", code, out)
	}
}

func Test0142_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	write(t, repo, map[string]string{"notes.md": misspelledNote})
	out, code := hookEvent(t, repo, editEvent(repo, "notes.md"))
	sees(t, out, code, 0, `"decision":"block"`, "fitness: cspell failed on notes.md", "[cspell] notes.md:1", "mispeled")
	write(t, repo, map[string]string{"pages/[id].md": misspelledNote, "pages/i.md": misspelledNote})
	out, code = hookEvent(t, repo, editEvent(repo, "pages/[id].md"))
	sees(t, out, code, 0, "[cspell] pages/[id].md:1")
	if strings.Contains(out, "pages/i.md") {
		t.Errorf("an edit to pages/[id].md judged pages/i.md too:\n%s", out)
	}
}

func Test0142_2(t *testing.T) {
	repo := example(t, "happyRepo", map[string]string{
		".fitnessrc.json": `{"policy": "external", "disabledChecks": ["requirements"]}` + "\n",
		"go.mod":          "module example.com/hooked\n\ngo 1.22\n",
		"hooked.go":       "// Package hooked holds one constant.\npackage hooked\n\n// Answer is the answer.\nconst Answer = 42\n",
		"hooked_test.go":  "package hooked\n\nimport \"testing\"\n\nfunc TestAnswer(t *testing.T) { t.Fatal(\"go-test ran\") }\n",
	})
	runner := filepath.Join(installOnly(t), "fitness")
	best := time.Hour
	for i := range 3 {
		file := fmt.Sprintf("note%d.md", i)
		write(t, repo, map[string]string{file: "A short note.\n"})
		raw, err := json.Marshal(map[string]any{"cwd": repo, "hook_event_name": "PostToolUse", "tool_name": "Edit", "tool_input": map[string]any{"file_path": filepath.Join(repo, file)}})
		mustDo(t, err)
		cmd := exec.Command(runner, "claude-hook")
		cmd.Dir, cmd.Env, cmd.Stdin = repo, userEnv(), strings.NewReader(string(raw))
		start := time.Now()
		out, err := cmd.CombinedOutput()
		best = min(best, time.Since(start))
		if err != nil || len(out) > 0 {
			t.Fatalf("the edit hook on a clean note must pass silently: %v\n%s", err, out)
		}
	}
	if best > 300*time.Millisecond {
		t.Errorf("the edit hook took %s at best; it must answer within 300ms", best)
	}
	out, code := hookEvent(t, repo, editEvent(repo, "hooked_test.go"))
	silentPass(t, out, code)
}

func Test0142_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	body := func(text string) string {
		return "gh pr create --title 'Add notes' --body \"$(cat <<'EOF'\n" + text + "EOF\n)\""
	}
	out, code := hookEvent(t, repo, bashEvent(body("Some **bold** text.\n")))
	sees(t, out, code, 0, `"permissionDecision":"deny"`, "failed on the pr body", "[markdown-no-bold-italic]")
	out, code = hookEvent(t, repo, bashEvent(body(cleanText)))
	silentPass(t, out, code)
	write(t, repo, map[string]string{"body.md": "Some **bold** text.\n", "sub/body.md": "Some **bold** text.\n"})
	for _, command := range []string{
		"cd sub && /opt/homebrew/bin/gh -R owner/repo pr create --title t -F body.md",
		"gh pr comment 5 --body-file - <<-EOF\n\tSome **bold** text.\n\tEOF\n",
		`gh pr create --title t --body='Some **bold** text.'`,
		`gh pr edit 5 -b "Some **bold** \"text\"."`,
		"gh pr create --title t --body-file - <<< 'Some **bold** text.'",
		"git push && gh pr create --title t -F body.md",
	} {
		out, code = hookEvent(t, repo, bashEvent(command))
		sees(t, out, code, 0, `"permissionDecision":"deny"`, "[markdown-no-bold-italic]")
	}
	comment := "cd sub && gh pr comment 5 --body-file - <<'EOF'\n```mermaid\nflowchart TB\n    a --> b\n```\nEOF\n"
	out, code = hookEvent(t, repo, bashEvent(comment))
	sees(t, out, code, 0, `"permissionDecision":"deny"`, "failed on the pr comment body", "[comment-structure]")
}

func Test0142_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	write(t, repo, map[string]string{"fresh.md": "Some **bold** text.\n"})
	for _, command := range []string{
		"ls -la && gh pr view 5 --json body",
		`gh pr create --title t --body "$BODY"`,
		"gh pr create --title t --body-file - <<EOF\nSee $(cat notes.md)\nEOF\n",
		"printf 'Fine.' > fresh.md && gh pr create --title t -F fresh.md",
	} {
		out, code := hookEvent(t, repo, bashEvent(command))
		silentPass(t, out, code)
	}
}

func Test0142_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	write(t, repo, map[string]string{"notes.md": misspelledNote, "café.md": misspelledNote})
	out, code := hookEvent(t, repo, stopEvent(false))
	sees(t, out, code, 0, `"decision":"block"`, "fitness: cspell failed on the changed files", "[cspell] notes.md:1", "[cspell] café.md:1")
	out, code = hookEvent(t, repo, stopEvent(true))
	silentPass(t, out, code)
}

func Test0142_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": ""})
	write(t, repo, map[string]string{"notes.md": misspelledNote})
	out, code := hookEvent(t, repo, editEvent(repo, "notes.md"))
	silentPass(t, out, code)
}

func Test0142_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	write(t, repo, map[string]string{"notes.md": misspelledNote})
	out, code := hookEvent(t, repo, editEvent(repo, "notes.md"))
	sees(t, out, code, 0, `"decision":"block"`)
	out, code = hookEvent(t, repo, editEvent(repo, "notes.md"))
	silentPass(t, out, code)
}

// claudeSettings runs `fitness-install --claude-settings` on file and
// returns the settings it leaves.
func claudeSettings(t *testing.T, file string) map[string]any {
	t.Helper()
	out, code := fiRun(t, t.TempDir(), nil, "--claude-settings", file)
	sees(t, out, code, 0, "fitness: Claude hooks in "+file+" run ")
	var settings map[string]any
	mustDo(t, json.Unmarshal([]byte(fiFile(t, file)), &settings))
	return settings
}

// fitnessHandlers lists, per event, the commands of the handlers that run
// `claude-hook`, and returns every other command too.
func fitnessHandlers(settings map[string]any) (ours map[string][]string, others []string) {
	ours = map[string][]string{}
	hooks, _ := settings["hooks"].(map[string]any)
	for event, groups := range hooks {
		for _, g := range groups.([]any) {
			for _, h := range g.(map[string]any)["hooks"].([]any) {
				command := h.(map[string]any)["command"].(string)
				if strings.HasSuffix(command, " claude-hook") {
					ours[event] = append(ours[event], command)
				} else {
					others = append(others, command)
				}
			}
		}
	}
	return ours, others
}

func Test0142_8(t *testing.T) {
	t.Parallel()
	installer(t)
	file := filepath.Join(t.TempDir(), "settings.json")
	mine := `{"theme": "dark", "hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "my-check"}]}]}}`
	mustDo(t, os.WriteFile(file, []byte(mine), 0o600))
	claudeSettings(t, file)
	settings := claudeSettings(t, file)
	if settings["theme"] != "dark" {
		t.Errorf("my theme setting was lost: %v", settings)
	}
	ours, others := fitnessHandlers(settings)
	if len(others) != 1 || others[0] != "my-check" {
		t.Errorf("my own hook must stay once, got %v", others)
	}
	for _, event := range []string{"PostToolUse", "PreToolUse", "Stop"} {
		installedOnce(t, event, ours[event])
	}
	keptMode(t, file, 0o600)
	skillDeclares(t, settings)
}

// keptMode asserts file still has mode.
func keptMode(t *testing.T, file string, mode os.FileMode) {
	t.Helper()
	if info, err := os.Stat(file); err != nil || info.Mode().Perm() != mode {
		t.Errorf("%s must keep its mode %o: %v", file, mode, err)
	}
}

// skillDeclares asserts the fitness skill's front matter ends with the hooks
// the installer wrote into settings: the same events, matchers and timeouts.
func skillDeclares(t *testing.T, settings map[string]any) {
	t.Helper()
	hooks := settings["hooks"].(map[string]any)
	want := "hooks:\n"
	for _, event := range []string{"PostToolUse", "PreToolUse", "Stop"} {
		want += "  " + event + ":\n" + declared(hooks[event].([]any))
	}
	skill := fiFile(t, filepath.Join("..", "..", "skills", "fitness", "SKILL.md"))
	if !strings.Contains(skill, "\n"+want+"---\n") {
		t.Errorf("the fitness skill front matter must end with these hooks:\n%s\ngot:\n%s", want, skill)
	}
}

// declared is the front matter YAML for the last group of an event, which
// the installer appends.
func declared(groups []any) string {
	group := groups[len(groups)-1].(map[string]any)
	handler := group["hooks"].([]any)[0].(map[string]any)
	yaml := "    - hooks:\n"
	if matcher, ok := group["matcher"].(string); ok {
		yaml = "    - matcher: \"" + matcher + "\"\n      hooks:\n"
	}
	return yaml + fmt.Sprintf("        - type: command\n          command: fitness claude-hook\n          timeout: %v\n", handler["timeout"])
}

// installedOnce asserts event holds one Fitness hook, run by the installed
// fitness.
func installedOnce(t *testing.T, event string, commands []string) {
	t.Helper()
	if len(commands) != 1 {
		t.Fatalf("%s must hold the Fitness hook once, got %v", event, commands)
	}
	runner := strings.TrimSuffix(commands[0], " claude-hook")
	if info, err := os.Stat(runner); err != nil || filepath.Base(runner) != "fitness" || info.IsDir() {
		t.Errorf("%s runs %q, not the installed fitness", event, runner)
	}
}
