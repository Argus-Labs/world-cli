package telemetry

import (
	"errors"
	"slices"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	sentryInitialized bool
)

// SentryHook is a custom hook that implements zerolog.Hook interface
type SentryHook struct{}

// Run is called for every log event and implements the zerolog.Hook interface
func (h SentryHook) Run(_ *zerolog.Event, level zerolog.Level, msg string) {
	shouldBeLogged := slices.Contains(h.Levels(), level)
	if sentryInitialized && shouldBeLogged {
		// Capture error message
		if level == zerolog.ErrorLevel || level == zerolog.FatalLevel || level == zerolog.PanicLevel {
			sentry.CaptureException(errors.New(msg))
		}

		// Capture warning message
		if level == zerolog.WarnLevel || level == zerolog.DebugLevel {
			sentry.CaptureMessage(msg)
		}
	}
}

// Levels returns the log levels that this hook should be triggered for
func (h SentryHook) Levels() []zerolog.Level {
	return []zerolog.Level{zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.DebugLevel,
		zerolog.PanicLevel, zerolog.WarnLevel}
}

// SentryInit initialize sentry
func SentryInit(sentryDsn string) {
	if sentryDsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:                sentryDsn,
			EnableTracing:      true,
			TracesSampleRate:   1.0,
			ProfilesSampleRate: 1.0,
			AttachStacktrace:   true,
		})
		if err != nil {
			log.Err(err).Msg("Cannot initialize sentry")
			return
		}

		sentryInitialized = true
	}
}

func SentryFlush() {
	if sentryInitialized {
		err := recover()
		if err != nil {
			sentry.CurrentHub().Recover(err)
		}

		// Flush buffered events before the program terminates.
		// Set the timeout to the maximum duration the program can afford to wait.
		sentry.Flush(5 * time.Second) //nolint:gomnd
		sentryInitialized = false
	}
}
