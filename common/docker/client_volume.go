package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/util"
	"pkg.world.dev/world-cli/tea/component/multispinner"
	"pkg.world.dev/world-cli/tea/style"
)

func (c *Client) processVolume(ctx context.Context, processType processType, volumeName string) error {
	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)
	p := util.NewTeaProgram(multispinner.CreateSpinner([]string{volumeName}, cancel))

	errChan := make(chan error, 1)

	go func() {
		p.Send(multispinner.ProcessState{
			State: processInitName[processType],
			Type:  "volume",
			Name:  volumeName,
		})

		volumes, err := c.client.VolumeList(ctx, volume.ListOptions{})
		if err != nil {
			p.Send(multispinner.ProcessState{
				Icon:   style.CrossIcon.Render(),
				Type:   "volume",
				Name:   volumeName,
				State:  processInitName[processType],
				Detail: err.Error(),
				Done:   true,
			})
			errChan <- eris.Wrap(err, "Failed to list volumes")
			return
		}

		volumeExist := false
		for _, volume := range volumes.Volumes {
			if volume.Name == volumeName {
				volumeExist = true
				break
			}
		}

		switch processType {
		case CREATE:
			if !volumeExist {
				_, err = c.client.VolumeCreate(ctx, volume.CreateOptions{Name: volumeName})
			}
		case REMOVE:
			if volumeExist {
				err = c.client.VolumeRemove(ctx, volumeName, true)
			}
		case START, STOP:
			err = eris.New(fmt.Sprintf("%s process type is not supported for volumes", processName[processType]))
		default:
			err = eris.New(fmt.Sprintf("Unknown process type: %d", processType))
		}

		if err != nil {
			p.Send(multispinner.ProcessState{
				Icon:   style.CrossIcon.Render(),
				Type:   "volume",
				Name:   volumeName,
				State:  processInitName[processType],
				Detail: err.Error(),
				Done:   true,
			})
			errChan <- eris.Wrapf(err, "Failed to %s volume %s", processName[processType], volumeName)
			return
		}

		p.Send(multispinner.ProcessState{
			Icon:  style.TickIcon.Render(),
			Type:  "volume",
			Name:  volumeName,
			State: processFinishName[processType],
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
