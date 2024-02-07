package main

import (
	"github.com/denisbrodbeck/machineid"
	ph "github.com/posthog/posthog-go"
	"log"
	"os"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/posthog"
	"time"

	"github.com/getsentry/sentry-go"
	"pkg.world.dev/world-cli/cmd/world/root"
)

// This variable will be overridden by ldflags during build
// Example : go build -ldflags "-X main.AppVersion=1.0.0 -X main.PosthogApiKey=<POSTHOG_API_KEY> -X main.SentryDsn=<SENTRY_DSN>"
var (
	AppVersion    string
	PosthogApiKey string
	SentryDsn     string
)

const (
	postInstallationEvent = "World CLI Installation"
	runningEvent          = "World CLI Running"
)

func init() {
	// Set default app version in case not provided by ldflags
	if AppVersion == "" {
		AppVersion = "dev"
	}
	root.AppVersion = AppVersion
}

func main() {
	// Sentry initialization
	if SentryDsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:                SentryDsn,
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

	// Posthog Initialization
	posthog.Init(PosthogApiKey)
	defer posthog.Close()

	// Obtain the machine ID
	machineID, err := machineid.ProtectedID("world-cli")
	if err != nil {
		logger.Error(err)
	}

	// Create capture event for posthog
	event := ph.Capture{
		DistinctId: machineID,
		Timestamp:  time.Now(),
		Properties: map[string]interface{}{
			"version": AppVersion,
			"command": os.Args,
		},
	}

	// Capture event post installation
	if len(os.Args) > 1 && os.Args[1] == "post-installation" {
		event.Event = postInstallationEvent
		posthog.CaptureEvent(event)
		return
	}

	// Capture event running
	event.Event = runningEvent
	posthog.CaptureEvent(event)

	root.Execute()
}
