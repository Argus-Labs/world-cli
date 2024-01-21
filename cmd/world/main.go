package main

import (
	"log"
	"time"

	"github.com/getsentry/sentry-go"

	"pkg.world.dev/world-cli/cmd/world/root"
)

func main() {
	// Sentry initialization
	DSN := "" // Input DSN here, you can get it from https://argus-labs.sentry.io/settings/projects/world-cli/keys/
	if DSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:                DSN,
			EnableTracing:      true,
			TracesSampleRate:   1.0,
			ProfilesSampleRate: 1.0,
			AttachStacktrace:   true,
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		// Handle panic
		defer func() {
			err := recover()
			if err != nil {
				sentry.CurrentHub().Recover(err)
			}

			// Flush buffered events before the program terminates.
			// Set the timeout to the maximum duration the program can afford to wait.
			sentry.Flush(time.Second * 5)
		}()
	}

	rootCmd := root.New()
	rootCmd.Execute()
}
