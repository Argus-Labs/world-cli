package root

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-version"
	"github.com/rotisserie/eris"
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

var CLI struct {
	Create       *CreateCmd  `cmd:"" group:"Getting Started:" help:"Create a new World Engine project"`
	Doctor       *DoctorCmd  `cmd:"" group:"Getting Started:" help:"Check your development environment"`
	kong.Plugins             // put this here so tools will be in the right place
	Version      *VersionCmd `cmd:"" group:"Additional Commands:" help:"Show the version of the CLI"`
	// Help    *root.HelpCmd    `cmd:"" default:"1" group:"Additional Commands:" help:"Show more detailed help"`
	Verbose bool `flag:"" short:"v" help:"Enable World CLI Debug logs"`
}

type HelpCmd struct {
}

func (c *HelpCmd) Run() error {
	printer.Infoln(style.CLIHeader("World CLI", "Create, manage, and deploy World Engine projects with ease"))
	_ = checkLatestVersion()
	return nil
}

// Release structure to hold the data of the latest release.
type Release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
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

// contextWithSigterm provides a context that automatically terminates when either the parent context is canceled or
// when a termination signal is received.
//
//nolint:unused // we will want to use this in the future I think
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
