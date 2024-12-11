// Package errors provides standardized error handling for the World CLI
package errors

import (
	"fmt"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/logging"
)

// Common error types
var (
	ErrInvalidConfig     = eris.New("invalid configuration")
	ErrDockerOperation   = eris.New("docker operation failed")
	ErrNetworkOperation  = eris.New("network operation failed")
	ErrFileOperation     = eris.New("file operation failed")
	ErrInvalidArgument   = eris.New("invalid argument")
	ErrDependencyMissing = eris.New("required dependency missing")
)

// WrapIf wraps an error with a message if the error is not nil
func WrapIf(err error, msg string) error {
	if err == nil {
		return nil
	}
	return eris.Wrap(err, msg)
}

// LogError logs an error with appropriate context using zerolog
func LogError(err error, msg string) {
	if err == nil {
		return
	}

	if eris.Is(err, ErrInvalidConfig) {
		logging.Error(msg, err, "category", "config")
	} else if eris.Is(err, ErrDockerOperation) {
		logging.Error(msg, err, "category", "docker")
	} else if eris.Is(err, ErrNetworkOperation) {
		logging.Error(msg, err, "category", "network")
	} else if eris.Is(err, ErrFileOperation) {
		logging.Error(msg, err, "category", "file")
	} else if eris.Is(err, ErrInvalidArgument) {
		logging.Error(msg, err, "category", "validation")
	} else if eris.Is(err, ErrDependencyMissing) {
		logging.Error(msg, err, "category", "dependency")
	} else {
		logging.Error(msg, err, "category", "unknown")
	}
}

// Errorf creates a new error with formatting
func Errorf(format string, args ...interface{}) error {
	return eris.New(fmt.Sprintf(format, args...))
}

// Is reports whether any error in err's tree matches target
func Is(err, target error) bool {
	return eris.Is(err, target)
}

// Wrap wraps an error with a message
func Wrap(err error, msg string) error {
	return eris.Wrap(err, msg)
}

// Unwrap returns the next error in err's chain
func Unwrap(err error) error {
	return eris.Unwrap(err)
}
