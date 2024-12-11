package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"

	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/config"
	"pkg.world.dev/world-cli/infrastructure/docker/operations"
	"pkg.world.dev/world-cli/infrastructure/docker/service"
	"pkg.world.dev/world-cli/logging"
)

const (
	START processType = iota
	STOP
	REMOVE
	CREATE
)

var (
	processName = map[processType]string{
		START:  "start",
		STOP:   "stop",
		REMOVE: "remove",
		CREATE: "create",
	}

	processInitName = map[processType]string{
		START:  "starting",
		STOP:   "stopping",
		REMOVE: "removing",
		CREATE: "creating",
	}

	processFinishName = map[processType]string{
		START:  "started",
		STOP:   "stopped",
		REMOVE: "removed",
		CREATE: "created",
	}
)

type Client struct {
	client    *client.Client
	cfg       *config.Config
	operations *operations.Manager
}

func NewClient(cfg *config.Config) (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, eris.Wrap(err, "Failed to create docker client")
	}

	// Set BuildkitSupport
	service.BuildkitSupport = checkBuildkitSupport(cli)

	return &Client{
		client:     cli,
		cfg:        cfg,
		operations: operations.NewManager(cli),
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Start(ctx context.Context, serviceBuilders ...service.Builder) error {
	// Create network and volume if they don't exist
	namespace := c.cfg.DockerEnv["CARDINAL_NAMESPACE"]
	if err := c.createNetworkIfNotExists(ctx, namespace); err != nil {
		return eris.Wrap(err, "Failed to create network")
	}

	if err := c.processVolume(ctx, CREATE, namespace); err != nil {
		return eris.Wrap(err, "Failed to create volume")
	}

	// Initialize services
	dockerServices := make([]service.Service, 0, len(serviceBuilders))
	for _, sb := range serviceBuilders {
		dockerServices = append(dockerServices, sb(c.cfg))
	}

	// Pull and build images
	if err := c.pullImages(ctx, dockerServices...); err != nil {
		return eris.Wrap(err, "Failed to pull images")
	}

	if c.cfg.Build {
		if err := c.buildImages(ctx, dockerServices...); err != nil {
			return eris.Wrap(err, "Failed to build images")
		}
	}

	// Start containers using operations manager
	for _, ds := range dockerServices {
		if err := c.operations.ServiceOperation(ctx, ds, func(op operations.ContainerOperation) error {
			return c.operations.StartContainer(ctx, op)
		}); err != nil {
			return eris.Wrapf(err, "Failed to start container %s", ds.Name)
		}
	}

	// Log containers if not detached
	if !c.cfg.Detach {
		c.logMultipleContainers(ctx, dockerServices...)
	}

	// Stop containers when context is done if not detached
	if !c.cfg.Detach {
		go func() {
			<-ctx.Done()
			err := c.Stop(context.Background(), serviceBuilders...)
			if err != nil {
				logging.Error("Failed to stop containers", err)
			}
		}()
	}

	return nil
}

func (c *Client) Stop(ctx context.Context, serviceBuilders ...service.Builder) error {
	// Initialize services
	dockerServices := make([]service.Service, 0, len(serviceBuilders))
	for _, sb := range serviceBuilders {
		dockerServices = append(dockerServices, sb(c.cfg))
	}

	// Stop containers using operations manager
	for _, ds := range dockerServices {
		if err := c.operations.ServiceOperation(ctx, ds, func(op operations.ContainerOperation) error {
			return c.operations.StopContainer(ctx, op)
		}); err != nil {
			return eris.Wrapf(err, "Failed to stop container %s", ds.Name)
		}
	}

	return nil
}

func (c *Client) Purge(ctx context.Context, serviceBuilders ...service.Builder) error {
	// Initialize services
	dockerServices := make([]service.Service, 0, len(serviceBuilders))
	for _, sb := range serviceBuilders {
		dockerServices = append(dockerServices, sb(c.cfg))
	}

	// Remove containers using operations manager
	for _, ds := range dockerServices {
		if err := c.operations.ServiceOperation(ctx, ds, func(op operations.ContainerOperation) error {
			return c.operations.RemoveContainer(ctx, op)
		}); err != nil {
			return eris.Wrapf(err, "Failed to remove container %s", ds.Name)
		}
	}

	// Remove volume
	if err := c.processVolume(ctx, REMOVE, c.cfg.DockerEnv["CARDINAL_NAMESPACE"]); err != nil {
		return err
	}

	return nil
}

func (c *Client) Restart(ctx context.Context, serviceBuilders ...service.Builder) error {
	if err := c.Stop(ctx, serviceBuilders...); err != nil {
		return eris.Wrap(err, "Failed to stop containers during restart")
	}

	if err := c.Start(ctx, serviceBuilders...); err != nil {
		return eris.Wrap(err, "Failed to start containers during restart")
	}

	return nil
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
