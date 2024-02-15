package main

import (
	"os"

	"github.com/rs/zerolog/log"

	"pkg.world.dev/world-cli/cmd/world/root"
	_ "pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/telemetry"
)

// This variable will be overridden by ldflags during build
// Example : go build -ldflags "-X main.AppVersion=1.0.0 -X main.PosthogApiKey=<POSTHOG_API_KEY> -X main.SentryDsn=<SENTRY_DSN>"
var (
	AppVersion    string
	PosthogApiKey string
	SentryDsn     string
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
	telemetry.SentryInit(SentryDsn)
	defer telemetry.SentryFlush()

	// Set logger sentry hook
	log.Logger = log.Logger.Hook(telemetry.SentryHook{})

	// Posthog Initialization
	telemetry.PosthogInit(PosthogApiKey)
	defer telemetry.PosthogClose()

	// Capture event post installation
	if len(os.Args) > 1 && os.Args[1] == "post-installation" {
		telemetry.PosthogCaptureEvent(AppVersion, telemetry.PostInstallationEvent)
		return
	}

	// Capture event running
	telemetry.PosthogCaptureEvent(AppVersion, telemetry.RunningEvent)

	root.Execute()
}
