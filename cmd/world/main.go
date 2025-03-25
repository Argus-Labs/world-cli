package main

import (
	"os"
	"runtime/debug"

	"github.com/rs/zerolog/log"

	"pkg.world.dev/world-cli/cmd/world/forge"
	"pkg.world.dev/world-cli/cmd/world/root"
	"pkg.world.dev/world-cli/common/globalconfig"
	"pkg.world.dev/world-cli/telemetry"

	_ "pkg.world.dev/world-cli/common/logger"
)

// This variable will be overridden by ldflags during build
// Example:
/*
	go build -ldflags "-X main.PosthogAPIKey=<POSTHOG_API_KEY>
							-X main.SentryDsn=<SENTRY_DSN>>"
*/
var (
	PosthogAPIKey string
	SentryDsn     string
)

func init() {
	env, version := getEnvAndVersion()
	root.AppVersion = version
	globalconfig.Env = env

	forge.InitForge()
}

func main() {
	// Sentry initialization
	telemetry.SentryInit(SentryDsn, globalconfig.Env, root.AppVersion)
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
		telemetry.PosthogCaptureEvent(root.AppVersion, telemetry.PostInstallationEvent)
		return
	}

	// Capture event running
	telemetry.PosthogCaptureEvent(root.AppVersion, telemetry.RunningEvent)

	root.Execute()
}

func getEnvAndVersion() (string, string) {
	env := "unknown env"
	version := "unknown version"

	// Get the environment and version from the build info
	info, ok := debug.ReadBuildInfo()

	// If the build info is not available, return the default values
	if !ok {
		return env, version
	}

	// If the version is "(devel)", return the default values
	if info.Main.Version == "(devel)" {
		version = "v0.0.1-dev"
		env = "DEV"
	} else {
		version = info.Main.Version
		env = "PROD"
	}

	// override env using env variable
	if os.Getenv("WORLD_CLI_ENV") != "" {
		env = os.Getenv("WORLD_CLI_ENV")
	}

	return env, version
}
