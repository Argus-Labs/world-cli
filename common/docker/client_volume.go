package docker

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/volume"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/tea/component/multispinner"
	"pkg.world.dev/world-cli/tea/style"
)

func (c *Client) createVolumeIfNotExists(ctx context.Context, volumeName string) error {
	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)
	p := tea.NewProgram(multispinner.CreateSpinner([]string{volumeName}, cancel))

	errChan := make(chan error, 1)

	go func() {
		p.Send(multispinner.ProcessState{
			State: "creating",
			Type:  "volume",
			Name:  volumeName,
		})

		volumes, err := c.client.VolumeList(ctx, volume.ListOptions{})
		if err != nil {
			p.Send(multispinner.ProcessState{
				Icon:   style.CrossIcon.Render(),
				Type:   "volume",
				Name:   volumeName,
				State:  "creating",
				Detail: err.Error(),
				Done:   true,
			})
			errChan <- eris.Wrap(err, "Failed to list volumes")
			return
		}

		volumeIsExist := false
		for _, volume := range volumes.Volumes {
			if volume.Name == volumeName {
				volumeIsExist = true
			}
		}

		if !volumeIsExist {
			_, err = c.client.VolumeCreate(ctx, volume.CreateOptions{Name: volumeName})
			if err != nil {
				p.Send(multispinner.ProcessState{
					Icon:   style.CrossIcon.Render(),
					Type:   "volume",
					Name:   volumeName,
					State:  "creating",
					Detail: err.Error(),
					Done:   true,
				})
				errChan <- eris.Wrapf(err, "Failed to create volume %s", volumeName)
				return
			}
		}

		p.Send(multispinner.ProcessState{
			Icon:  style.TickIcon.Render(),
			Type:  "volume",
			Name:  volumeName,
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

func (c *Client) removeVolume(ctx context.Context, volumeName string) error {
	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)
	p := tea.NewProgram(multispinner.CreateSpinner([]string{volumeName}, cancel))

	errChan := make(chan error, 1)

	go func() {
		p.Send(multispinner.ProcessState{
			State: "removing",
			Type:  "volume",
			Name:  volumeName,
		})

		volumes, err := c.client.VolumeList(ctx, volume.ListOptions{})
		if err != nil {
			p.Send(multispinner.ProcessState{
				Icon:   style.CrossIcon.Render(),
				Type:   "volume",
				Name:   volumeName,
				State:  "removing",
				Detail: err.Error(),
				Done:   true,
			})
			errChan <- eris.Wrap(err, "Failed to list volumes")
			return
		}

		isExist := false
		for _, v := range volumes.Volumes {
			if v.Name == volumeName {
				isExist = true
				break
			}
		}

		// Remove the volume if it exists
		if isExist {
			err = c.client.VolumeRemove(ctx, volumeName, true)
			if err != nil {
				p.Send(multispinner.ProcessState{
					Icon:   style.CrossIcon.Render(),
					Type:   "volume",
					Name:   volumeName,
					State:  "removing",
					Detail: err.Error(),
					Done:   true,
				})
				errChan <- eris.Wrapf(err, "Failed to remove volume %s", volumeName)
				return
			}
		}

		p.Send(multispinner.ProcessState{
			Icon:  style.TickIcon.Render(),
			Type:  "volume",
			Name:  volumeName,
			State: "removed",
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
