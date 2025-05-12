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
	"github.com/docker/docker/client"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/world/forge"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/tea/component/multispinner"
	"pkg.world.dev/world-cli/tea/style"
)

type processType int

func (c *Client) processMultipleContainers(
	ctx context.Context,
	processType processType,
	services ...service.Service,
) error {
	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup error channel and program
	errChan := make(chan error, len(services))
	p := c.setupProcessProgram(services, cancel)

	// Launch processing goroutines
	c.launchProcessGoroutines(ctx, processType, services, p, errChan)

	// Run the program
	if _, err := p.Run(); err != nil {
		return eris.Wrap(err, "Failed to run multispinner")
	}

	// Collect and process errors
	return c.collectProcessErrors(errChan)
}

func (c *Client) setupProcessProgram(services []service.Service, cancel context.CancelFunc) *tea.Program {
	dockerServicesNames := make([]string, len(services))
	for i, dockerService := range services {
		dockerServicesNames[i] = dockerService.Name
	}
	return forge.NewTeaProgram(multispinner.CreateSpinner(dockerServicesNames, cancel))
}

func (c *Client) launchProcessGoroutines(
	ctx context.Context,
	processType processType,
	services []service.Service,
	p *tea.Program,
	errChan chan error,
) {
	for _, ds := range services {
		dockerService := ds
		go c.processSingleContainer(ctx, processType, dockerService, p, errChan)
	}
}

func (c *Client) processSingleContainer(
	ctx context.Context,
	processType processType,
	dockerService service.Service,
	p *tea.Program,
	errChan chan error,
) {
	c.sendProcessInitState(p, dockerService.Name, processType)

	err := c.executeProcessType(ctx, processType, dockerService)
	if err != nil {
		c.sendProcessErrorState(p, dockerService.Name, processType, err)
		errChan <- err
		return
	}

	c.sendProcessSuccessState(p, dockerService.Name, processType)
}

func (c *Client) executeProcessType(ctx context.Context, processType processType, dockerService service.Service) error {
	switch processType {
	case STOP:
		return c.stopContainer(ctx, dockerService.Name)
	case REMOVE:
		return c.removeContainer(ctx, dockerService.Name)
	case START:
		return c.startContainer(ctx, dockerService)
	case CREATE:
		return eris.New("CREATE process type is not supported for containers")
	default:
		return eris.New(fmt.Sprintf("Unknown process type: %d", processType))
	}
}

func (c *Client) sendProcessInitState(p *tea.Program, name string, processType processType) {
	p.Send(multispinner.ProcessState{
		Icon:  style.CrossIcon.Render(),
		Type:  "container",
		Name:  name,
		State: processInitName[processType],
	})
}

func (c *Client) sendProcessErrorState(p *tea.Program, name string, processType processType, err error) {
	p.Send(multispinner.ProcessState{
		Icon:   style.CrossIcon.Render(),
		Type:   "container",
		Name:   name,
		State:  processInitName[processType],
		Detail: err.Error(),
		Done:   true,
	})
}

func (c *Client) sendProcessSuccessState(p *tea.Program, name string, processType processType) {
	p.Send(multispinner.ProcessState{
		Icon:  style.TickIcon.Render(),
		Type:  "container",
		Name:  name,
		State: processFinishName[processType],
		Done:  true,
	})
}

func (c *Client) collectProcessErrors(errChan chan error) error {
	close(errChan)
	errs := make([]error, 0)
	for err := range errChan {
		errs = append(errs, err)
	}

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
						printer.Infof("Error logging container %s: %v. Reattaching...\n", id, err)
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
			printer.Infof("[%s] %s", style.ForegroundPrint(containerID, colors[styleNumber]), cleanLog)
		case 2: //nolint:gomnd // Stderr
			// TODO: what content should be printed for stderr?
			printer.Infof("[%s] %s", style.ForegroundPrint(containerID, colors[styleNumber]), cleanLog)
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
