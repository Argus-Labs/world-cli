package posthog

import (
	ph "github.com/posthog/posthog-go"

	"pkg.world.dev/world-cli/common/logger"
)

var (
	client      ph.Client
	initialized bool
)

// Init Posthog initialization
func Init(posthogApiKey string) {
	if posthogApiKey != "" {
		client = ph.New(posthogApiKey)
		initialized = true
	}
}

func CaptureEvent(capture ph.Capture) {
	if initialized {
		err := client.Enqueue(capture)
		if err != nil {
			logger.Error(err)
		}
	}
}

func Close() {
	if initialized {
		err := client.Close()
		if err != nil {
			logger.Error(err)
		}
	}
}
