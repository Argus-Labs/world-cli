package docker

import (
	"context"

	"github.com/docker/docker/api/types/network"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/util"
	"pkg.world.dev/world-cli/tea/component/multispinner"
	"pkg.world.dev/world-cli/tea/style"
)

func (c *Client) createNetworkIfNotExists(ctx context.Context, networkName string) error {
	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)
	p := util.NewTeaProgram(multispinner.CreateSpinner([]string{networkName}, cancel))

	errChan := make(chan error, 1)

	go func() {
		p.Send(multispinner.ProcessState{
			State: "creating",
			Type:  "network",
			Name:  networkName,
		})

		networks, err := c.client.NetworkList(ctx, network.ListOptions{})
		if err != nil {
			p.Send(multispinner.ProcessState{
				Icon:   style.CrossIcon.Render(),
				Type:   "network",
				Name:   networkName,
				State:  "creating",
				Detail: err.Error(),
				Done:   true,
			})
			errChan <- eris.Wrap(err, "Failed to list networks")
			return
		}

		networkExist := false
		for _, network := range networks {
			if network.Name == networkName {
				networkExist = true
				break
			}
		}

		if !networkExist {
			_, err = c.client.NetworkCreate(ctx, networkName, network.CreateOptions{
				Driver: "bridge",
			})
			if err != nil {
				p.Send(multispinner.ProcessState{
					Icon:   style.CrossIcon.Render(),
					Type:   "network",
					Name:   networkName,
					State:  "creating",
					Detail: err.Error(),
					Done:   true,
				})
				errChan <- eris.Wrapf(err, "Failed to create network %s", networkName)
				return
			}
		}

		p.Send(multispinner.ProcessState{
			Icon:  style.TickIcon.Render(),
			Type:  "network",
			Name:  networkName,
			State: "created",
			Done:  true,
		})
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		return eris.Wrap(err, "Failed to run multispinner")
	}

	// Close the error channel and check for errors
	close(errChan)
	if err := <-errChan; err != nil {
		return err
	}

	return nil
}
