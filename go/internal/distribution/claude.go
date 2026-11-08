package distribution

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// ClaudeHook is one Claude Code hook that runs `fitness claude-hook`: the
// event it answers, the tools it matches, and its timeout in seconds.
type ClaudeHook struct {
	Event   string
	Matcher string
	Timeout int
}

// stopMargin is how much longer than both totals the Stop hook waits, for
// the handshakes before the checks and the report after them.
const stopMargin = 30 * time.Second

// ClaudeHooks are the hooks the installer and the fitness skill's front
// matter both register; each event's job is in `fitness claude-hook`.
var ClaudeHooks = []ClaudeHook{
	{Event: "PostToolUse", Matcher: "Edit|Write|MultiEdit|NotebookEdit", Timeout: 30},
	{Event: "PreToolUse", Matcher: "Bash|mcp__.*", Timeout: 30},
	{Event: "Stop", Timeout: int((checkkit.FastTotal + checkkit.TestsTotal + stopMargin) / time.Second)},
	{Event: "SubagentStop", Timeout: 300},
	{Event: "SessionStart", Timeout: 30},
	{Event: "SessionEnd", Timeout: 30},
}

// claudeHookArg is the runner subcommand every hook runs, which also names
// the entries this installer owns in a settings file.
const claudeHookArg = "claude-hook"

// WriteClaudeSettings adds the Fitness hooks, run by the bundle's fitness,
// to each Claude Code settings file. It replaces the entries an earlier
// install wrote and keeps every other setting.
func WriteClaudeSettings(bundle string, files []string, log io.Writer) error {
	command := shellQuote(filepath.Join(bundle, "fitness")) + " " + claudeHookArg
	for _, file := range files {
		if err := writeClaudeSettings(file, command); err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		fmt.Fprintf(log, "fitness: Claude hooks in %s run %s\n", file, command)
	}
	return nil
}

func writeClaudeSettings(file, command string) error {
	if real, err := filepath.EvalSymlinks(file); err == nil {
		file = real
	}
	settings, mode, err := readSettings(file)
	if err != nil {
		return err
	}
	raw, err := withFitnessHooks(settings["hooks"], command)
	if err != nil {
		return err
	}
	settings["hooks"] = raw
	return writeSettings(file, settings, mode)
}

// withFitnessHooks returns the hooks setting with the Fitness hooks once
// each, run by command, and every other hook as it was.
func withFitnessHooks(raw json.RawMessage, command string) (json.RawMessage, error) {
	hooks := map[string][]any{}
	if raw != nil && json.Unmarshal(raw, &hooks) != nil {
		return nil, errors.New("hooks is not an object of event lists")
	}
	for event, groups := range hooks {
		hooks[event] = withoutFitness(groups)
	}
	for _, hook := range ClaudeHooks {
		hooks[hook.Event] = append(hooks[hook.Event], hook.group(command))
	}
	maps.DeleteFunc(hooks, func(_ string, groups []any) bool { return len(groups) == 0 })
	return json.Marshal(hooks)
}

// group is the settings entry for one hook.
func (hook ClaudeHook) group(command string) map[string]any {
	group := map[string]any{"hooks": []any{map[string]any{"type": "command", "command": command, "timeout": hook.Timeout}}}
	if hook.Matcher != "" {
		group["matcher"] = hook.Matcher
	}
	return group
}

// withoutFitness drops the handlers that run `fitness claude-hook`, and any
// group they leave empty, keeping every other entry as written.
func withoutFitness(groups []any) []any {
	var kept []any
	for _, g := range groups {
		if group, ok := withoutFitnessHandlers(g); ok {
			kept = append(kept, group)
		}
	}
	return kept
}

// withoutFitnessHandlers drops the Fitness handlers from one group, and
// reports whether anything is left of it.
func withoutFitnessHandlers(g any) (any, bool) {
	group, _ := g.(map[string]any)
	handlers, isList := group["hooks"].([]any)
	if !isList {
		return g, true
	}
	var others []any
	for _, h := range handlers {
		if !isFitnessHandler(h) {
			others = append(others, h)
		}
	}
	group["hooks"] = others
	return group, len(others) > 0
}

// isFitnessHandler reports whether a handler runs `fitness claude-hook`.
func isFitnessHandler(h any) bool {
	handler, _ := h.(map[string]any)
	command, _ := handler["command"].(string)
	fields := strings.Fields(command)
	return len(fields) >= 2 && fields[len(fields)-1] == claudeHookArg && strings.HasSuffix(strings.Trim(fields[len(fields)-2], `'"`), "fitness")
}

// readSettings reads a settings file's top-level keys and mode; a missing
// or empty file is empty settings.
func readSettings(file string) (map[string]json.RawMessage, os.FileMode, error) {
	settings := map[string]json.RawMessage{}
	info, err := os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		return settings, 0o644, nil
	}
	data, readErr := os.ReadFile(file)
	if err = cmp.Or(err, readErr); err != nil {
		return nil, 0, err
	}
	if len(strings.TrimSpace(string(data))) > 0 && json.Unmarshal(data, &settings) != nil {
		return nil, 0, errors.New("not a JSON object")
	}
	return settings, info.Mode().Perm(), nil
}

// writeSettings writes settings through a temporary file beside it, so a
// failed write leaves the old file whole.
func writeSettings(file string, settings map[string]json.RawMessage, mode os.FileMode) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(file), ".settings-*.json")
	if err != nil {
		return err
	}
	_, writeErr := tmp.Write(append(data, '\n'))
	if err := errors.Join(writeErr, tmp.Chmod(mode), tmp.Close()); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), file)
}

// shellQuote quotes path for a hook command when it holds a character the
// shell would read.
func shellQuote(path string) string {
	if !strings.ContainsAny(path, " \t\n'\"\\$`;&|<>()*?[]#~") {
		return path
	}
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
