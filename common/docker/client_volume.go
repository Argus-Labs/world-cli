package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/tea/style"
)

func (c *Client) createVolumeIfNotExists(ctx context.Context, volumeName string) error {
	volumes, err := c.client.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return err
	}

	for _, volume := range volumes.Volumes {
		if volume.Name == volumeName {
			logger.Debugf("Volume %s already exists\n", volumeName)
			return nil
		}
	}

	_, err = c.client.VolumeCreate(ctx, volume.CreateOptions{Name: volumeName})
	if err != nil {
		return err
	}

	fmt.Printf("Created volume %s\n", volumeName)
	return nil
}

func (c *Client) removeVolume(ctx context.Context, volumeName string) error {
	volumes, err := c.client.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return eris.Wrap(err, "Failed to list volumes")
	}

	isExist := false
	for _, v := range volumes.Volumes {
		if v.Name == volumeName {
			isExist = true
			break
		}
	}

	// Return if volume does not exist
	if !isExist {
		return nil
	}

	contextPrint("Removing", "1", "volume", volumeName)

	err = c.client.VolumeRemove(ctx, volumeName, true)
	if err != nil {
		return eris.Wrapf(err, "Failed to remove volume %s", volumeName)
	}

	fmt.Println(style.TickIcon.Render())
	return nil
}
