package docker

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/container"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/infrastructure/docker/service"
	"pkg.world.dev/world-cli/tea/component/multispinner"
	"pkg.world.dev/world-cli/tea/style"
)

func (c *Client) logMultipleContainers(ctx context.Context, services ...service.Service) {
	var wg sync.WaitGroup

	// Start logging output for each container
	for i, dockerService := range services {
		wg.Add(1)
		go func(id string, index int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					err := c.logContainerOutput(ctx, id, index)
					if err != nil && !errors.Is(err, context.Canceled) {
						fmt.Printf("Error logging container %s: %v. Reattaching...\n", id, err)
						time.Sleep(2 * time.Second) //nolint:gomnd // Sleep for 2 seconds before reattaching
					}
				}
			}
		}(dockerService.Name, i)
	}

	// Wait for all logging goroutines to finish
	wg.Wait()
}

func (c *Client) logContainerOutput(ctx context.Context, containerID string, styleNumber int) error {
	colors := []string{
		"#00FF00", // Green
		"#0000FF", // Blue
		"#00FFFF", // Cyan
		"#FF00FF", // Magenta
		"#FFA500", // Orange
		"#800080", // Purple
		"#FFC0CB", // Pink
		"#87CEEB", // Sky Blue
		"#32CD32", // Lime Green
	}

	// Create options for logs
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}

	// Fetch logs from the container
	out, err := c.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return eris.Wrapf(err, "failed to get logs for container %s", containerID)
	}
	defer out.Close()

	reader := bufio.NewReader(out)
	for {
		// Read the 8-byte header
		header := make([]byte, 8) //nolint:gomnd // 8 bytes
		if _, err := io.ReadFull(reader, header); err != nil {
			if err == io.EOF {
				break
			}
			return eris.Wrapf(err, "failed to read log header for container %s", containerID)
		}

		// Determine the stream type from the first byte
		streamType := header[0]
		// Get the size of the log payload from the last 4 bytes
		size := binary.BigEndian.Uint32(header[4:])

		// Read the log payload based on the size
		payload := make([]byte, size)
		if _, err := io.ReadFull(reader, payload); err != nil {
			if err == io.EOF {
				break
			}
			return eris.Wrapf(err, "failed to read log payload for container %s", containerID)
		}

		// Clean the log message by removing ANSI escape codes
		cleanLog := removeFirstAnsiEscapeCode(string(payload))

		// Print the cleaned log message with container ID and color
		coloredID := style.ForegroundPrint(containerID, colors[styleNumber%len(colors)])
		if streamType == 1 { // Stdout
			fmt.Printf("[%s] %s", coloredID, cleanLog)
		} else if streamType == 2 { // Stderr
			fmt.Printf("[%s] %s", coloredID, cleanLog)
		}
	}

	return nil
}

// removeFirstAnsiEscapeCode removes only the first ANSI escape code from a string
func removeFirstAnsiEscapeCode(input string) string {
	re := regexp.MustCompile(`^\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(input, "")
}
