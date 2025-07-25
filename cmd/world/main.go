package main

import (
	"context"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/lipgloss"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog/log"
	"pkg.world.dev/world-cli/internal/app/world-cli/clients/api"
	"pkg.world.dev/world-cli/internal/app/world-cli/clients/browser"
	"pkg.world.dev/world-cli/internal/app/world-cli/clients/repo"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/cardinal"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/cloud"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/evm"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/organization"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/project"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/root"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/user"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/config"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/telemetry"
	cmdsetup "pkg.world.dev/world-cli/internal/app/world-cli/controllers/cmd_setup"
	cfgService "pkg.world.dev/world-cli/internal/app/world-cli/services/config"
	"pkg.world.dev/world-cli/internal/app/world-cli/services/input"
	"pkg.world.dev/world-cli/internal/pkg/logger"
	"pkg.world.dev/world-cli/internal/pkg/printer"
)

const (
	// For local development.
	worldForgeBaseURLLocal = "http://localhost:8001"
	// For Argus Dev.
	worldForgeBaseURLDev = "https://forge.argus.dev"
	// For Argus Production.
	worldForgeBaseURLProd = "https://forge.world.dev"
	// For local development.
	worldForgeRPCBaseURLLocal = "http://localhost:8002/rpc"
	// RPC Dev URL.
	worldForgeRPCBaseURLDev = "https://rpc.argus.dev"
	// RPC Prod URL.
	worldForgeRPCBaseURLProd = "https://rpc.world.dev"
	// For Argus ID Dev.
	argusIDBaseURLDev = "https://id.argus.dev"
	// For Argus ID Production.
	argusIDBaseURLProd = "https://id.argus.gg"
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

func main() {
	// Create a channel to receive signals.
	sigChan := make(chan os.Signal, 1)

	// Notify the channel when specific signals are received.
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start a goroutine to handle signals.
	go func() {
		// Block until a signal is received.
		sig := <-sigChan
		switch sig {
		case os.Interrupt, syscall.SIGTERM:
			os.Exit(0)
		}
	}()

	env, version := getEnvAndVersion()

	// Sentry initialization
	telemetry.SentryInit(SentryDsn, env, version)
	defer telemetry.SentryFlush()

	// Set up config directory "~/.worldcli/"
	err := config.SetupCLIConfigDir()
	if err != nil {
		log.Err(err).Msg("could not setup config folder")
		return
	}

	// Posthog Initialization
	telemetry.PosthogInit(PosthogAPIKey)
	defer telemetry.PosthogClose()

	// Capture event post installation
	if len(os.Args) > 1 && os.Args[1] == "post-installation" {
		telemetry.PosthogCaptureEvent(version, telemetry.PostInstallationEvent)
		return
	}

	// Capture event running
	telemetry.PosthogCaptureEvent(version, telemetry.RunningEvent)

	// Initialize dependencies once for all commands
	dependencies, err := initDependencies(env, version)
	if err != nil {
		log.Err(err).Msg("could not initialize dependencies")
		return
	}

	// Initialize packages
	CLI.Plugins = kong.Plugins{
		&CardinalCmdPlugin,
		&CloudCmdPlugin,
		&EvmCmdPlugin,
		&ProjectCmdPlugin,
		&OrganizationCmdPlugin,
		&UserCmdPlugin,
	}

	ctx := kong.Parse(
		&CLI,
		kong.Name("world"),
		kong.Description("World CLI: Your complete toolkit for World Engine development"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)

	// Set verbose mode if the flag is enabled
	if CLI.Verbose {
		logger.VerboseMode = true
	}

	realCtx := contextWithSigterm(context.Background())
	SetKongParentsAndContext(realCtx, dependencies, &CLI)
	SetKongParentsAndContext(realCtx, dependencies, &CardinalCmdPlugin)
	SetKongParentsAndContext(realCtx, dependencies, &CloudCmdPlugin)
	SetKongParentsAndContext(realCtx, dependencies, &EvmCmdPlugin)
	SetKongParentsAndContext(realCtx, dependencies, &ProjectCmdPlugin)
	SetKongParentsAndContext(realCtx, dependencies, &OrganizationCmdPlugin)
	SetKongParentsAndContext(realCtx, dependencies, &UserCmdPlugin)
	err = ctx.Run()
	if err != nil {
		sentry.CaptureException(err)
		if logger.VerboseMode {
			logger.Errors(err)
		}
		// TODO: Remove this once we have a proper error handling
		printer.Errorln(err.Error())
	}
	// print log stack
	logger.PrintLogs()
	printer.NewLine(1)
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
		env = "LOCAL"
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

// SetKongParents recursively sets Parent pointers for Kong CLI structs.
func SetKongParentsAndContext(ctx context.Context, dependencies cmdsetup.Dependencies, cmd interface{}) {
	setParentsAndContext(ctx, dependencies, reflect.ValueOf(cmd), reflect.ValueOf(nil))
}

//nolint:gocognit // this does exactly what it's supposed to do
func setParentsAndContext(
	ctx context.Context,
	dependencies cmdsetup.Dependencies,
	v reflect.Value,
	parent reflect.Value,
) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	t := v.Type()
	for i := range v.NumField() {
		field := v.Field(i)
		fieldType := t.Field(i)
		// Set Parent pointer if field is named "Parent" and is a pointer
		if fieldType.Name == "Parent" && field.Kind() == reflect.Ptr && parent.IsValid() {
			if field.IsNil() {
				field.Set(parent)
				continue
			}
		}
		// Set Context pointer if field is named "Context" and is a pointer
		if fieldType.Name == "Context" && field.Kind() == reflect.Interface {
			if field.IsNil() {
				field.Set(reflect.ValueOf(ctx))
				continue
			}
		}
		// Inject Dependencies if field is named "Dependencies"
		if fieldType.Name == "Dependencies" && fieldType.Type.String() == "cmdsetup.Dependencies" {
			if field.CanSet() {
				field.Set(reflect.ValueOf(dependencies))
				continue
			}
		}
		// Recurse into subcommands (fields with `cmd:""` tag)
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			_, ok := fieldType.Tag.Lookup("cmd")
			if ok {
				setParentsAndContext(ctx, dependencies, field, v.Addr())
			}
		}
	}
}

// contextWithSigterm provides a context that automatically terminates when either the parent context is canceled or
// when a termination signal is received.
func contextWithSigterm(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	go func() {
		defer cancel()

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

		select {
		case <-signalCh:
			printer.Infoln(textStyle.Render("Interrupt signal received. Terminating..."))
		case <-ctx.Done():
			printer.Infoln(textStyle.Render("Cancellation signal received. Terminating..."))
		}
	}()

	return ctx
}

func initDependencies(env, version string) (cmdsetup.Dependencies, error) {
	var baseURL string
	var rpcURL string
	var argusIDBaseURL string
	switch env {
	case cfgService.EnvLocal:
		baseURL = worldForgeBaseURLLocal
		rpcURL = worldForgeRPCBaseURLLocal
		argusIDBaseURL = argusIDBaseURLDev
		printer.Notificationln("Forge Env: LOCAL")
	case cfgService.EnvDev:
		baseURL = worldForgeBaseURLDev
		rpcURL = worldForgeRPCBaseURLDev
		argusIDBaseURL = argusIDBaseURLDev
		printer.Notificationln("Forge Env: DEV")
	default:
		baseURL = worldForgeBaseURLProd
		rpcURL = worldForgeRPCBaseURLProd
		argusIDBaseURL = argusIDBaseURLProd
	}

	configService, err := cfgService.NewService(env)
	if err != nil {
		return cmdsetup.Dependencies{}, err
	}

	repoClient := repo.NewClient()
	inputService := input.NewService()
	// TODO: Make a proper RPC client
	apiClient := api.NewClient(baseURL, rpcURL, argusIDBaseURL)
	apiClient.SetAuthToken(configService.GetConfig().Credential.Token)

	projectHandler := project.NewHandler(
		repoClient,
		configService,
		apiClient,
		&inputService,
	)

	orgHandler := organization.NewHandler(
		projectHandler,
		&inputService,
		apiClient,
		configService,
	)

	userHandler := user.NewHandler(
		apiClient,
		&inputService,
	)

	cloudHandler := cloud.NewHandler(
		apiClient,
		configService,
		projectHandler,
		&inputService,
	)

	evmHandler := evm.NewHandler()

	cardinalHandler := cardinal.NewHandler()

	setupController := cmdsetup.NewController(
		configService,
		repoClient,
		orgHandler,
		projectHandler,
		apiClient,
		&inputService,
	)

	browserClient := browser.NewClient()
	rootHandler := root.NewHandler(version, configService, apiClient, setupController, browserClient)

	return cmdsetup.Dependencies{
		ConfigService:       configService,
		InputService:        &inputService,
		APIClient:           apiClient,
		RepoClient:          repoClient,
		OrganizationHandler: orgHandler,
		ProjectHandler:      projectHandler,
		UserHandler:         userHandler,
		CloudHandler:        cloudHandler,
		EVMHandler:          evmHandler,
		SetupController:     setupController,
		RootHandler:         rootHandler,
		CardinalHandler:     cardinalHandler,
	}, nil
}
