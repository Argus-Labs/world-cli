package root

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/printer"
)

const (
	// latestReleaseURL is the URL to fetch the latest release of the CLI.
	latestReleaseURL = "https://github.com/Argus-Labs/world-cli/releases/latest"
	// httpTimeout is the timeout for the HTTP client.
	httpTimeout = 2 * time.Second
)

func (h *Handler) SetAppVersion(version string) {
	h.AppVersion = version
}

func (h *Handler) Version(check bool) error {
	printer.Infof("World CLI %s\n", h.AppVersion)
	if check {
		return h.checkLatestVersion()
	}
	return nil
}

func (h *Handler) checkLatestVersion() error {
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

	if h.AppVersion != "" {
		currentVersion, err := version.NewVersion(h.AppVersion)
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
