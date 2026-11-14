package checkkit

import (
	"errors"
	"fmt"
)

// internalError marks an environment error, such as a missing tool or
// unreachable service, rather than a repo finding.
type internalError struct{ err error }

func (e *internalError) Error() string { return e.err.Error() }

func (e *internalError) Unwrap() error { return e.err }

// Internal marks err as an internal error, reported as info with exit 3.
// A nil err stays nil.
func Internal(err error) error {
	if err == nil {
		return nil
	}
	return &internalError{err: err}
}

// IsInternal reports whether err, or an error it wraps, is internal.
func IsInternal(err error) bool {
	var target *internalError
	return errors.As(err, &target)
}

// ToolMissing is the internal error for a tool the check needs but cannot find.
func ToolMissing(name string) error {
	return Internal(fmt.Errorf("%s is not installed or not on PATH", name))
}
