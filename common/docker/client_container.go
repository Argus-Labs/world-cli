package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/tea/style"
)

func (c *Client) startContainer(ctx context.Context, service service.Service) error {
	contextPrint("Starting", "2", "container", service.Name)

	// Check if the container exists
	exist, err := c.containerExists(ctx, service.Name)
	if err != nil {
		return eris.Wrapf(err, "Failed to check if container %s exists", service.Name)
	} else if !exist {
		// Create the container if it does not exist
		_, err := c.client.ContainerCreate(ctx, &service.Config, &service.HostConfig,
			&service.NetworkingConfig, &service.Platform, service.Name)
		if err != nil {
			fmt.Println(style.CrossIcon.Render())
			return err
		}
	}

	// Start the container
	if err := c.client.ContainerStart(ctx, service.Name, container.StartOptions{}); err != nil {
		fmt.Println(style.CrossIcon.Render())
		return err
	}

	fmt.Println(style.TickIcon.Render())
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
	contextPrint("Stopping", "1", "container", containerName)

	// Check if the container exists
	exist, err := c.containerExists(ctx, containerName)
	if !exist {
		fmt.Println(style.TickIcon.Render())
		return err
	}

	// Stop the container
	err = c.client.ContainerStop(ctx, containerName, container.StopOptions{})
	if err != nil {
		fmt.Println(style.CrossIcon.Render())
		return eris.Wrapf(err, "Failed to stop container %s", containerName)
	}

	fmt.Println(style.TickIcon.Render())
	return nil
}

func (c *Client) removeContainer(ctx context.Context, containerName string) error {
	contextPrint("Removing", "1", "container", containerName)

	// Check if the container exists
	exist, err := c.containerExists(ctx, containerName)
	if !exist {
		fmt.Println(style.TickIcon.Render())
		return err
	}

	// Stop the container
	err = c.client.ContainerStop(ctx, containerName, container.StopOptions{})
	if err != nil {
		fmt.Println(style.CrossIcon.Render())
		return eris.Wrapf(err, "Failed to stop container %s", containerName)
	}

	// Remove the container
	err = c.client.ContainerRemove(ctx, containerName, container.RemoveOptions{})
	if err != nil {
		fmt.Println(style.CrossIcon.Render())
		return eris.Wrapf(err, "Failed to remove container %s", containerName)
	}

	fmt.Println(style.TickIcon.Render())
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
					fmt.Printf("Stopping logging for container %s: %v\n", id, ctx.Err())
					return
				default:
					err := c.logContainerOutput(ctx, id, strconv.Itoa(i))
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

func (c *Client) logContainerOutput(ctx context.Context, containerID, style string) error {
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

	// Print logs
	buf := make([]byte, 4096) //nolint:gomnd
	for {
		n, err := out.Read(buf)
		if n > 0 {
			fmt.Printf("[%s] %s", foregroundPrint(containerID, style), buf[:n])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	return nil
}
