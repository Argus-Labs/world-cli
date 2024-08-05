package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/logger"
)

type Client struct {
	client *client.Client
	cfg    *config.Config
}

func NewClient(cfg *config.Config) (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, eris.Wrap(err, "Failed to create docker client")
	}

	// Set BuildkitSupport
	service.BuildkitSupport = checkBuildkitSupport(cli)

	return &Client{
		client: cli,
		cfg:    cfg,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Start(ctx context.Context, cfg *config.Config,
	serviceBuilders ...service.Builder) error {
	defer func() {
		if !cfg.Detach {
			err := c.Stop(cfg, serviceBuilders...)
			if err != nil {
				logger.Error("Failed to stop containers", err)
			}
		}
	}()

	namespace := cfg.DockerEnv["CARDINAL_NAMESPACE"]
	err := c.createNetworkIfNotExists(ctx, namespace)
	if err != nil {
		return eris.Wrap(err, "Failed to create network")
	}

	err = c.createVolumeIfNotExists(ctx, namespace)
	if err != nil {
		return eris.Wrap(err, "Failed to create volume")
	}

	// get all services
	dockerServices := make([]service.Service, 0)
	for _, sb := range serviceBuilders {
		dockerServices = append(dockerServices, sb(cfg))
	}

	// Pull all images before starting containers
	err = c.pullImages(ctx, dockerServices...)
	if err != nil {
		return eris.Wrap(err, "Failed to pull images")
	}

	// Start all containers
	for _, dockerService := range dockerServices {
		// build image if needed
		if cfg.Build && dockerService.Dockerfile != "" {
			if err := c.buildImage(ctx, dockerService.Dockerfile, dockerService.BuildTarget, dockerService.Image); err != nil {
				return eris.Wrap(err, "Failed to build image")
			}
		}

		// create container & start
		if err := c.startContainer(ctx, dockerService); err != nil {
			return eris.Wrap(err, "Failed to create container")
		}
	}

	// log containers if not detached
	if !cfg.Detach {
		c.logMultipleContainers(ctx, dockerServices...)
	}

	return nil
}

func (c *Client) Stop(cfg *config.Config, serviceBuilders ...service.Builder) error {
	ctx := context.Background()
	for _, sb := range serviceBuilders {
		dockerService := sb(cfg)
		if err := c.stopContainer(ctx, dockerService.Name); err != nil {
			return eris.Wrap(err, "Failed to stop container")
		}
	}

	return nil
}

func (c *Client) Purge(cfg *config.Config, serviceBuilders ...service.Builder) error {
	ctx := context.Background()
	for _, sb := range serviceBuilders {
		dockerService := sb(cfg)
		if err := c.removeContainer(ctx, dockerService.Name); err != nil {
			return eris.Wrap(err, "Failed to remove container")
		}
	}

	err := c.removeVolume(ctx, cfg.DockerEnv["CARDINAL_NAMESPACE"])
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Restart(ctx context.Context, cfg *config.Config,
	serviceBuilders ...service.Builder) error {
	// stop containers
	err := c.Stop(cfg, serviceBuilders...)
	if err != nil {
		return err
	}

	return c.Start(ctx, cfg, serviceBuilders...)
}

func (c *Client) Exec(ctx context.Context, containerID string, cmd []string) (string, error) {
	// Create Exec Instance
	exec := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execIDResp, err := c.client.ContainerExecCreate(ctx, containerID, exec)
	if err != nil {
		return "", eris.Wrapf(err, "Failed to create exec instance")
	}

	// Start Exec Instance
	resp, err := c.client.ContainerExecAttach(ctx, execIDResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", eris.Wrapf(err, "Failed to start exec instance")
	}
	defer resp.Close()

	// Read and demultiplex the output
	var outputBuf bytes.Buffer
	header := make([]byte, 8) //nolint:gomnd

	for {
		_, err := io.ReadFull(resp.Reader, header)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", eris.Wrapf(err, "Failed to read exec output")
		}

		stream := header[0]
		size := binary.BigEndian.Uint32(header[4:8])

		if stream == 1 { // stdout
			if _, err := io.CopyN(&outputBuf, resp.Reader, int64(size)); err != nil {
				return "", eris.Wrapf(err, "Failed to read stdout")
			}
		} else {
			// Skip stderr or other streams
			if _, err := io.CopyN(io.Discard, resp.Reader, int64(size)); err != nil {
				return "", eris.Wrapf(err, "Failed to read stderr")
			}
		}
	}

	// Return the output as a string
	return outputBuf.String(), nil
}
