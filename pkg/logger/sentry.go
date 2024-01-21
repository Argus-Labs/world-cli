package logger

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

// SentryHook is a custom hook that implements zerolog.Hook interface
type SentryHook struct{}

// Run is called for every log event and implements the zerolog.Hook interface
func (h SentryHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Capture error message
	sentry.CaptureException(fmt.Errorf(msg))
}

// Levels returns the log levels that this hook should be triggered for
func (h SentryHook) Levels() []zerolog.Level {
	return []zerolog.Level{zerolog.ErrorLevel, zerolog.FatalLevel}
}
