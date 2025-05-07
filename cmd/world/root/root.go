package root

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/getsentry/sentry-go"
	"github.com/hashicorp/go-version"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/cmd/world/cardinal"
	"pkg.world.dev/world-cli/cmd/world/evm"
	"pkg.world.dev/world-cli/cmd/world/forge"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/tea/style"
)

const (
	// latestReleaseURL is the URL to fetch the latest release of the CLI.
	latestReleaseURL = "https://github.com/Argus-Labs/world-cli/releases/latest"
	// httpTimeout is the timeout for the HTTP client.
	httpTimeout = 2 * time.Second
)

// rootCmd represents the base command.
// Usage: `world`.
var rootCmd = &cobra.Command{
	Use:   "world",
	Short: "Your complete toolkit for World Engine development",
	Long:  style.CLIHeader("World CLI", "Create, manage, and deploy World Engine projects with ease"),
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		return checkLatestVersion()
	},
}
var RootCmdTesting = rootCmd

// Release structure to hold the data of the latest release.
type Release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// CmdInit initializes the root command.
func CmdInit() {
	// Enable case-insensitive commands
	cobra.EnableCaseInsensitive = true //nolint:reassign // intentionally setting cobra global config as designed

	// Disable printing usage help text when command returns a non-nil error
	rootCmd.SilenceUsage = true

	// Injects a context that is canceled when a sigterm signal is received
	rootCmd.SetContext(contextWithSigterm(context.Background()))

	// Register groups
	rootCmd.AddGroup(&cobra.Group{ID: "starter", Title: "Getting Started:"})
	rootCmd.AddGroup(&cobra.Group{ID: "core", Title: "Tools:"})

	// Register base commands
	doctorCmd := getDoctorCmd(os.Stdout)
	createCmd := getCreateCmd(os.Stdout)
	rootCmd.AddCommand(createCmd, doctorCmd, versionCmd)

	// Register subcommands
	rootCmd.AddCommand(cardinal.BaseCmd)
	rootCmd.AddCommand(evm.BaseCmd)

	// Register forge command
	forge.AddCommands(rootCmd)

	// Remove completion subcommand
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// Execute adds all child commands to the root command and sets flags appropriately.
// It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		sentry.CaptureException(err)
		logger.Errors(err)
	}
	// print log stack
	logger.PrintLogs()
}

func checkLatestVersion() error {
	// Create a request with the context
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return eris.Wrap(err, "error creating request")
	}

	// Create a new HTTP client and execute the request
	client := &http.Client{
		Timeout: httpTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			// Return an error to prevent following redirects
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Debug(eris.Wrap(err, "error fetching the latest release"))
		return nil
	}
	defer resp.Body.Close()

	// Check if the status code is 302
	// GitHub responds with a 302 redirect to the actual latest release URL, which contains the version number
	if resp.StatusCode != http.StatusFound {
		logger.Debug(eris.Wrap(eris.New("status is not 302"), "error fetching the latest release"))
		return nil
	}

	// Get the latest release URL from the response
	redirectURL := resp.Header.Get("Location")
	// Get the latest release version from the URL
	latestReleaseVersion := strings.TrimPrefix(redirectURL, "https://github.com/Argus-Labs/world-cli/releases/tag/")

	if AppVersion != "" {
		currentVersion, err := version.NewVersion(AppVersion)
		if err != nil {
			return eris.Wrap(err, "error parsing current version")
		}

		latestVersion, err := version.NewVersion(latestReleaseVersion)
		if err != nil {
			return eris.Wrap(err, "error parsing latest version")
		}

		if currentVersion.LessThan(latestVersion) {
			printer.NewLine(1)
			printer.Notificationf("New version %s is available!", latestVersion.String())

			printer.Notificationln("To update, run: go install pkg.world.dev/world-cli/cmd/world@latest")
		}
	}
	return nil
}

func CheckLatestVersionTesting() error {
	return checkLatestVersion()
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
