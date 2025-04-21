package telemetry

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog/log"
)

var (
	sentryInitialized bool
)

// SentryInit initialize sentry.
func SentryInit(sentryDsn string, env string, appVersion string) {
	if sentryDsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:                sentryDsn,
			EnableTracing:      true,
			TracesSampleRate:   1.0,
			ProfilesSampleRate: 1.0,
			AttachStacktrace:   true,
			Environment:        env,
			Release:            fmt.Sprintf("world-cli@%s", appVersion),
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
