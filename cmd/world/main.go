package main

import (
	"os"

	"github.com/rs/zerolog/log"

	"pkg.world.dev/world-cli/cmd/world/root"
	"pkg.world.dev/world-cli/common/globalconfig"
	"pkg.world.dev/world-cli/telemetry"

	_ "pkg.world.dev/world-cli/common/logger"
)

// This variable will be overridden by ldflags during build
// Example:
/*
	go build -ldflags "-X main.AppVersion=1.0.0 -X main.PosthogAPIKey=<POSTHOG_API_KEY>
							-X main.SentryDsn=<SENTRY_DSN> -X main.Env=<DEV|PROD>"
*/
var (
	AppVersion    string
	PosthogAPIKey string
	SentryDsn     string
	Env           string
)

func init() {
	// Set default app version in case not provided by ldflags
	if AppVersion == "" {
		AppVersion = "v0.0.1-dev"
	}
	root.AppVersion = AppVersion

	if Env == "" {
		Env = "DEV"
	}
	globalconfig.Env = Env
}

func main() {
	// Sentry initialization
	telemetry.SentryInit(SentryDsn, Env, AppVersion)
	defer telemetry.SentryFlush()

	// Set up config directory "~/.worldcli/"
	err := globalconfig.SetupConfigDir()
	if err != nil {
		log.Err(err).Msg("could not setup config folder")
		return
	}

	// Posthog Initialization
	telemetry.PosthogInit(PosthogAPIKey)
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
