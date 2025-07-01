package browser

import (
	"os/exec"
	"runtime"

	"pkg.world.dev/world-cli/common/printer"
)

// OpenURL opens the given URL in the default browser.
func (c *Client) OpenURL(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		printer.Infof("Could not automatically open browser. Please visit this URL:\n%s\n", url)
	}
	if err != nil {
		printer.Infof("Failed to open browser automatically. Please visit this URL:\n%s\n", url)
	}
	return nil
}
