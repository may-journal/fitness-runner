package checkkit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Outcome is what one tool run left behind. Code is the exit code, or 1
// when the process ended without one, so a caller that only reads Code sees
// any failure. SpawnErr and Signaled tell a broken environment apart from a
// tool that ran and judged.
type Outcome struct {
	Code     int
	Stdout   string
	Stderr   string
	SpawnErr error
	Signaled bool
	// Err is the error the run returned, nil on exit 0.
	Err error
}

// Exec runs cmd with stdout and stderr captured apart. A writer the caller
// already set on cmd.Stderr, such as os.Stderr for live output, still
// receives the stream.
func Exec(cmd *exec.Cmd) Outcome {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = tee(cmd.Stderr, &stderr)
	o := outcome(cmd.Run())
	o.Stdout, o.Stderr = stdout.String(), stderr.String()
	return o
}

// ExecCombined runs cmd with stdout and stderr in one buffer, in the order
// the tool wrote them, and returns it as Stdout.
func ExecCombined(cmd *exec.Cmd) Outcome {
	var combined bytes.Buffer
	cmd.Stdout, cmd.Stderr = &combined, &combined
	o := outcome(cmd.Run())
	o.Stdout = combined.String()
	return o
}

func tee(w io.Writer, buf *bytes.Buffer) io.Writer {
	if w == nil {
		return buf
	}
	return io.MultiWriter(w, buf)
}

// outcome classifies a finished run's error.
func outcome(err error) Outcome {
	o := Outcome{Err: err}
	if err == nil {
		return o
	}
	o.Code = 1
	if errors.Is(err, exec.ErrWaitDelay) {
		return o
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		o.SpawnErr = err
		return o
	}
	if exitErr.ExitCode() > 0 {
		o.Code = exitErr.ExitCode()
	} else {
		o.Signaled = true
	}
	return o
}

// EnvErr is the internal error for a run that never judged: the tool could
// not start, or a signal ended it. Nil when the tool ran to an exit code.
func (o Outcome) EnvErr(name string) error {
	switch {
	case o.SpawnErr != nil:
		return Internal(fmt.Errorf("%s could not start: %w", name, o.SpawnErr))
	case o.Signaled:
		return Internal(fmt.Errorf("%s was stopped by a signal: %w", name, o.Err))
	}
	return nil
}

// sandboxText is output a tool prints when its environment, such as a
// sandbox without network or write access, stopped it rather than the code.
var sandboxText = []string{"dial tcp", "permission denied", "operation not permitted", "no required module"}

// SandboxText returns the first sandbox phrase in output, matched without
// case, or empty when there is none.
func SandboxText(output string) string {
	lower := strings.ToLower(output)
	for _, phrase := range sandboxText {
		if strings.Contains(lower, phrase) {
			return phrase
		}
	}
	return ""
}
