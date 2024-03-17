package telemetry

import (
	"os"
	"path/filepath"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/posthog/posthog-go"
	"github.com/rs/zerolog/log"
)

const (
	PostInstallationEvent = "World CLI Installation"
	RunningEvent          = "World CLI Running"
	timestampFile         = ".worldcli"
)

var (
	posthogClient      posthog.Client
	posthogInitialized bool
	lastLoggedTime     time.Time
)

// Init Posthog initialization
func PosthogInit(posthogAPIKey string) {
	if posthogAPIKey != "" {
		posthogClient = posthog.New(posthogAPIKey)
		posthogInitialized = true

		// get last logged time
		lastTime, err := getLastLoggedTime()
		if err != nil {
			log.Err(err).Msg("Cannot get last logged time")
		}

		lastLoggedTime = lastTime

		// Update last visited timestamp
		err = updateLastLoggedTime(time.Now())
		if err != nil {
			log.Err(err).Msg("Cannot update last logged time")
		}
	}
}

// getLastLoggedTime reads the last visited timestamp from the file.
func getLastLoggedTime() (time.Time, error) {
	filePath, err := getTimestampFilePath()
	if err != nil {
		return time.Time{}, err
	}

	if _, err = os.Stat(filePath); os.IsNotExist(err) {
		// Return a zero time if the file does not exist
		return time.Time{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return time.Time{}, err
	}

	timestamp, err := time.Parse(time.DateOnly, string(data))
	if err != nil {
		return time.Time{}, err
	}

	return timestamp, nil
}

// getTimestampFilePath returns the path to the timestamp file.
func getTimestampFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, timestampFile), nil
}

// updateLastLoggedTime updates the last visited timestamp in the file.
func updateLastLoggedTime(timestamp time.Time) error {
	filePath, err := getTimestampFilePath()
	if err != nil {
		return err
	}

	data := []byte(timestamp.Format(time.DateOnly))

	return os.WriteFile(filePath, data, 0644) //nolint:gosec // not applicable
}

// isSameDay checks if two timestamps are from the same day.
func isSameDay(time1, time2 time.Time) bool {
	return time1.Year() == time2.Year() &&
		time1.Month() == time2.Month() &&
		time1.Day() == time2.Day()
}

func PosthogCaptureEvent(context, event string) {
	if posthogInitialized && (!isSameDay(lastLoggedTime, time.Now()) || event != RunningEvent) {
		// Obtain the machine ID
		machineID, err := machineid.ProtectedID("world-cli")
		if err != nil {
			log.Err(err).Msg("Cannot get machine id")
			return
		}

		// Capture the event
		err = posthogClient.Enqueue(posthog.Capture{
			DistinctId: machineID,
			Timestamp:  time.Now(),
			Event:      event,
			Properties: map[string]interface{}{
				"context": context,
			},
		})
		if err != nil {
			log.Err(err).Msg("Cannot capture event")
		}
	}
}

func PosthogClose() {
	if posthogInitialized {
		err := posthogClient.Close()
		if err != nil {
			log.Err(err).Msg("Cannot close posthog client")
		}
		posthogInitialized = false
	}
}
