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

func (c *Client) Build(ctx context.Context,
	pushTo string,
	pushAuth string,
	serviceBuilders ...service.Builder) error {
	namespace := c.cfg.DockerEnv["CARDINAL_NAMESPACE"]

	err := c.processVolume(ctx, CREATE, namespace)
	if err != nil {
		return eris.Wrap(err, "Failed to create volume")
	}

	// get all services
	dockerServices := make([]service.Service, 0)
	for _, sb := range serviceBuilders {
		ds := sb(c.cfg)
		dockerServices = append(dockerServices, ds)
	}

	// Pull all images before starting containers
	err = c.pullImages(ctx, dockerServices...)
	if err != nil {
		return eris.Wrap(err, "Failed to pull images")
	}

	// Build all images before starting containers
	err = c.buildImages(ctx, dockerServices...)
	if err != nil {
		return eris.Wrap(err, "Failed to build images")
	}

	if pushTo != "" {
		err := c.pushImages(ctx, pushTo, pushAuth, dockerServices...)
		if err != nil {
			return eris.Wrap(err, "Failed to push images")
		}
	}
	return nil
}

func (c *Client) Start(ctx context.Context,
	serviceBuilders ...service.Builder) error {
	defer func() {
		if !c.cfg.Detach {
			err := c.Stop(context.Background(), serviceBuilders...)
			if err != nil {
				logger.Error("Failed to stop containers", err)
			}
		}
	}()

	namespace := c.cfg.DockerEnv["CARDINAL_NAMESPACE"]
	err := c.createNetworkIfNotExists(ctx, namespace)
	if err != nil {
		return eris.Wrap(err, "Failed to create network")
	}

	err = c.processVolume(ctx, CREATE, namespace)
	if err != nil {
		return eris.Wrap(err, "Failed to create volume")
	}

	// get all services
	dockerServices := make([]service.Service, 0)
	for _, sb := range serviceBuilders {
		ds := sb(c.cfg)
		dockerServices = append(dockerServices, ds)
	}

	// Pull all images before starting containers
	err = c.pullImages(ctx, dockerServices...)
	if err != nil {
		return eris.Wrap(err, "Failed to pull images")
	}

	// Build all images before starting containers
	if c.cfg.Build {
		err = c.buildImages(ctx, dockerServices...)
		if err != nil {
			return eris.Wrap(err, "Failed to build images")
		}
	}

	// Start all containers
	err = c.processMultipleContainers(ctx, START, dockerServices...)
	if err != nil {
		return eris.Wrap(err, "Failed to start containers")
	}

	// log containers if not detached
	if !c.cfg.Detach {
		c.logMultipleContainers(ctx, dockerServices...)
	}

	return nil
}

func (c *Client) Stop(ctx context.Context, serviceBuilders ...service.Builder) error {
	// get all services
	dockerServices := make([]service.Service, 0)
	for _, sb := range serviceBuilders {
		ds := sb(c.cfg)
		dockerServices = append(dockerServices, ds)
	}

	// Stop all containers
	err := c.processMultipleContainers(ctx, STOP, dockerServices...)
	if err != nil {
		return eris.Wrap(err, "Failed to stop containers")
	}

	return nil
}

func (c *Client) Purge(ctx context.Context, serviceBuilders ...service.Builder) error {
	// get all services
	dockerServices := make([]service.Service, 0)
	for _, sb := range serviceBuilders {
		ds := sb(c.cfg)
		dockerServices = append(dockerServices, ds)
	}

	// remove all containers
	err := c.processMultipleContainers(ctx, REMOVE, dockerServices...)
	if err != nil {
		return eris.Wrap(err, "Failed to remove containers")
	}

	err = c.processVolume(ctx, REMOVE, c.cfg.DockerEnv["CARDINAL_NAMESPACE"])
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Restart(ctx context.Context,
	serviceBuilders ...service.Builder) error {
	// stop containers
	err := c.Stop(ctx, serviceBuilders...)
	if err != nil {
		return err
	}

	return c.Start(ctx, serviceBuilders...)
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
