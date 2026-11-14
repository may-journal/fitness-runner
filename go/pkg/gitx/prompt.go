package gitx

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// batchSSH is the ssh command git runs when the user set none, failing instead
// of asking for a password.
const batchSSH = "ssh -o BatchMode=yes"

// NoPrompt returns the settings that make git, ssh and gh fail instead of
// waiting for a password, keeping the user's own ssh command.
func NoPrompt(dir string) []string {
	env := []string{"GIT_TERMINAL_PROMPT=0", "GH_PROMPT_DISABLED=1"}
	if !ownSSH(dir) {
		env = append(env, "GIT_SSH_COMMAND="+batchSSH)
	}
	return env
}

// ownSSH reports whether the user chose the ssh command git runs.
func ownSSH(dir string) bool {
	for _, name := range []string{"GIT_SSH_COMMAND", "GIT_SSH"} {
		if os.Getenv(name) != "" {
			return true
		}
	}
	out, err := exec.Command("git", "-C", dir, "config", "--get", "core.sshCommand").Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

// DisablePrompts sets NoPrompt's settings in this process's environment, so
// every git, ssh and gh it starts fails instead of waiting.
func DisablePrompts(dir string) {
	for _, kv := range NoPrompt(dir) {
		name, value, _ := strings.Cut(kv, "=")
		_ = os.Setenv(name, value)
	}
}

// refusedPrompt matches the line git or ssh prints when a password prompt is
// refused, naming the remote URL or host.
var refusedPrompt = regexp.MustCompile(`for '([^']+)': terminal prompts disabled|(\S+): Permission denied \(`)

// RefusedPrompt returns the remote a refused password prompt in text names,
// and whether text holds one.
func RefusedPrompt(text string) (string, bool) {
	m := refusedPrompt.FindStringSubmatch(text)
	if m == nil {
		return "", false
	}
	return m[1] + m[2], true
}

// PromptNote says that remote asked for a password a check never waits for.
func PromptNote(remote string) string {
	return remote + " asked for a password, and checks never wait for one: save a credential or key for it"
}
