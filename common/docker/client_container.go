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

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/world/forge"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/tea/component/multispinner"
	"pkg.world.dev/world-cli/tea/style"
)

type processType int

func (c *Client) processMultipleContainers(ctx context.Context, processType processType,
	services ...service.Service) error {
	// Collect the names of the services
	dockerServicesNames := make([]string, len(services))
	for i, dockerService := range services {
		dockerServicesNames[i] = dockerService.Name
	}

	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)

	// Channel to collect errors from the goroutines
	errChan := make(chan error, len(dockerServicesNames))

	// Create tea program for multispinner
	p := forge.NewTeaProgram(multispinner.CreateSpinner(dockerServicesNames, cancel))

	// Process all containers
	for _, ds := range services {
		// capture the dockerService
		dockerService := ds

		go func() {
			p.Send(multispinner.ProcessState{
				Icon:  style.CrossIcon.Render(),
				Type:  "container",
				Name:  dockerService.Name,
				State: processInitName[processType],
			})

			// call the respective function based on the process type
			var err error
			switch processType {
			case STOP:
				err = c.stopContainer(ctx, dockerService.Name)
			case REMOVE:
				err = c.removeContainer(ctx, dockerService.Name)
			case START:
				err = c.startContainer(ctx, dockerService)
			case CREATE:
				err = eris.New("CREATE process type is not supported for containers")
			default:
				err = eris.New(fmt.Sprintf("Unknown process type: %d", processType))
			}

			if err != nil {
				p.Send(multispinner.ProcessState{
					Icon:   style.CrossIcon.Render(),
					Type:   "container",
					Name:   dockerService.Name,
					State:  processInitName[processType],
					Detail: err.Error(),
					Done:   true,
				})
				errChan <- err
				return
			}

			// if no error, send success
			p.Send(multispinner.ProcessState{
				Icon:  style.TickIcon.Render(),
				Type:  "container",
				Name:  dockerService.Name,
				State: processFinishName[processType],
				Done:  true,
			})
		}()
	}

	// Run the program
	if _, err := p.Run(); err != nil {
		return eris.Wrap(err, "Failed to run multispinner")
	}

	// Close the error channel and check for errors
	close(errChan)
	errs := make([]error, 0)
	for err := range errChan {
		errs = append(errs, err)
	}

	// If there were any errors, return them as a combined error
	if len(errs) > 0 {
		return eris.New(fmt.Sprintf("Errors: %v", errs))
	}

	return nil
}

func (c *Client) startContainer(ctx context.Context, service service.Service) error {
	// Check if the container exists
	exist, err := c.containerExists(ctx, service.Name)
	if err != nil {
		return eris.Wrapf(err, "Failed to check if container %s exists", service.Name)
	} else if !exist {
		// Create the container if it does not exist
		_, err := c.client.ContainerCreate(ctx, &service.Config, &service.HostConfig,
			&service.NetworkingConfig, &service.Platform, service.Name)
		if err != nil {
			return err
		}
	}

	// Start the container
	if err := c.client.ContainerStart(ctx, service.Name, container.StartOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *Client) containerExists(ctx context.Context, containerName string) (bool, error) {
	_, err := c.client.ContainerInspect(ctx, containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, eris.Wrapf(err, "Failed to inspect container %s", containerName)
	}

	return true, nil
}

func (c *Client) stopContainer(ctx context.Context, containerName string) error {
	// Check if the container exists
	exist, err := c.containerExists(ctx, containerName)
	if !exist {
		return err
	}

	// Stop the container
	err = c.client.ContainerStop(ctx, containerName, container.StopOptions{
		Signal: "SIGINT",
	})
	if err != nil {
		return eris.Wrapf(err, "Failed to stop container %s", containerName)
	}

	return nil
}

func (c *Client) removeContainer(ctx context.Context, containerName string) error {
	// Check if the container exists
	exist, err := c.containerExists(ctx, containerName)
	if !exist {
		return err
	}

	// Stop the container
	err = c.client.ContainerStop(ctx, containerName, container.StopOptions{})
	if err != nil {
		return eris.Wrapf(err, "Failed to stop container %s", containerName)
	}

	// Remove the container
	err = c.client.ContainerRemove(ctx, containerName, container.RemoveOptions{})
	if err != nil {
		return eris.Wrapf(err, "Failed to remove container %s", containerName)
	}

	return nil
}

func (c *Client) logMultipleContainers(ctx context.Context, services ...service.Service) {
	var wg sync.WaitGroup

	// Start logging output for each container
	for i, dockerService := range services {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					err := c.logContainerOutput(ctx, id, i)
					if err != nil && !errors.Is(err, context.Canceled) {
						fmt.Printf("Error logging container %s: %v. Reattaching...\n", id, err)
						time.Sleep(2 * time.Second) //nolint:gomnd // Sleep for 2 seconds before reattaching
					}
				}
			}
		}(dockerService.Name)
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
		return err
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
			return err
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
			return err
		}

		// Clean the log message by removing ANSI escape codes
		cleanLog := removeFirstAnsiEscapeCode(string(payload))

		// Print the cleaned log message
		switch streamType {
		case 1: // Stdout
			// TODO: what content should be printed for stdout?
			fmt.Printf("[%s] %s", style.ForegroundPrint(containerID, colors[styleNumber]), cleanLog)
		case 2: //nolint:gomnd // Stderr
			// TODO: what content should be printed for stderr?
			fmt.Printf("[%s] %s", style.ForegroundPrint(containerID, colors[styleNumber]), cleanLog)
		}
	}

	return nil
}

// Function to remove only the first ANSI escape code from a string.
func removeFirstAnsiEscapeCode(input string) string {
	ansiEscapePattern := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

	loc := ansiEscapePattern.FindStringIndex(input) // Find the first occurrence of an ANSI escape code
	if loc == nil {
		return input // If no ANSI escape code is found, return the input as-is
	}

	// Remove the first ANSI escape code by slicing out the matched part
	return input[:loc[0]] + input[loc[1]:]
}
