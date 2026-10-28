package gitx

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// batchSSH is the ssh command git runs when the user set none: BatchMode
// makes ssh fail instead of asking for a password or passphrase.
const batchSSH = "ssh -o BatchMode=yes"

// NoPrompt returns the settings that make git, ssh and gh fail at once
// instead of waiting for a password: GIT_TERMINAL_PROMPT=0,
// GH_PROMPT_DISABLED=1, and a batch-mode GIT_SSH_COMMAND. The ssh command is
// left out when the user set GIT_SSH_COMMAND, GIT_SSH or, for the repo at
// dir, core.sshCommand, so their own command still runs. A credential helper
// or password helper program is untouched and still answers.
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
// every git, ssh and gh it starts fails at once instead of waiting for a
// password.
func DisablePrompts(dir string) {
	for _, kv := range NoPrompt(dir) {
		name, value, _ := strings.Cut(kv, "=")
		_ = os.Setenv(name, value)
	}
}

// refusedPrompt matches the line git or ssh prints when it would have asked
// for a password but may not: git's names the remote URL, ssh's the host.
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
