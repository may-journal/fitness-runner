package distribution

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const hookConfigName = "fitness-hook.json"

func InstallHook(options Options) error {
	command := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-path", "hooks/pre-commit")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("find Git hook directory: %w", err)
	}
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	return writeHook(strings.TrimSpace(string(output)), binary, options)
}

func writeHook(path, binary string, options Options) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		return err
	}
	return publishHook(path, data, options)
}

func publishHook(path string, binary []byte, options Options) error {
	options.InstallHook = false
	options.GitHubAction = false
	config, err := json.Marshal(options)
	if err != nil {
		return err
	}
	// Reserve the hook before creating its config; never replace another owner.
	hook, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return fmt.Errorf("preserve existing hook %s: %w", path, err)
	}
	defer hook.Close()
	if err := finishHook(hook, config, binary); err != nil {
		os.Remove(path)
		return err
	}
	return nil
}

func finishHook(hook *os.File, config, binary []byte) error {
	path := filepath.Join(filepath.Dir(hook.Name()), hookConfigName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(config); err != nil {
		os.Remove(path)
		return err
	}
	if _, err := hook.Write(binary); err != nil {
		os.Remove(path)
		return err
	}
	return nil
}

func ReadHookOptions() (Options, error) {
	var options Options
	path, err := os.Executable()
	if err != nil {
		return options, err
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(path), hookConfigName))
	if err != nil {
		return options, err
	}
	err = json.Unmarshal(data, &options)
	return options, err
}
