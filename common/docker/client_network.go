package docker

import (
	"context"

	"github.com/docker/docker/api/types/network"

	"pkg.world.dev/world-cli/common/logger"
)

func (c *Client) createNetworkIfNotExists(ctx context.Context, networkName string) error {
	networks, err := c.client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return err
	}

	for _, network := range networks {
		if network.Name == networkName {
			logger.Infof("Network %s already exists", networkName)
			return nil
		}
	}

	_, err = c.client.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
	})
	if err != nil {
		return err
	}

	return nil
}
