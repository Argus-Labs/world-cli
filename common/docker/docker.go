package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
)

func Start(cfg *config.Config, configurations ...GetDockerConfig) error { //nolint:gocognit
	ctx := context.Background()

	cli, err := getCli()
	if err != nil {
		return eris.Wrap(err, "Failed to create docker client")
	}
	defer func() {
		if !cfg.Detach {
			err := Stop(cfg, configurations...)
			if err != nil {
				logger.Error("Failed to stop containers", err)
			}
		}

		err := cli.Close()
		if err != nil {
			logger.Error("Failed to close docker client", err)
		}
	}()

	namespace := cfg.DockerEnv["CARDINAL_NAMESPACE"]
	err = createNetworkIfNotExists(cli, namespace)
	if err != nil {
		return eris.Wrap(err, "Failed to create network")
	}

	err = createVolumeIfNotExists(cli, namespace)
	if err != nil {
		return eris.Wrap(err, "Failed to create volume")
	}

	// var for storing container names
	containers := make([]string, 0)

	// iterate over configurations and create containers
	for _, c := range configurations {
		configuration := c(cfg)
		if configuration.Dockerfile == nil {
			if err := pullImageIfNotExists(ctx, cli, configuration.Image); err != nil {
				return eris.Wrap(err, "Failed to pull image")
			}
		} else if cfg.Build {
			if err := buildImage(ctx, cli, *configuration.Dockerfile, configuration.Image); err != nil {
				return eris.Wrap(err, "Failed to build image")
			}
		}

		if err := createContainer(ctx, cli, configuration); err != nil {
			return eris.Wrap(err, "Failed to create container")
		}

		containers = append(containers, configuration.Name)
	}

	// log containers if not detached
	if !cfg.Detach {
		logContainers(cli, containers)
	}

	return nil
}

func Stop(cfg *config.Config, configurations ...GetDockerConfig) error {
	cli, err := getCli()
	if err != nil {
		return eris.Wrap(err, "Failed to create docker client")
	}
	defer func() {
		err := cli.Close()
		if err != nil {
			logger.Error("Failed to close docker client", err)
		}
	}()

	ctx := context.Background()
	for _, c := range configurations {
		configuration := c(cfg)
		if err := stopAndRemoveContainer(ctx, cli, configuration.Name); err != nil {
			return eris.Wrap(err, "Failed to stop container")
		}
	}

	err = removeNetwork(ctx, cli, cfg.DockerEnv["CARDINAL_NAMESPACE"])
	if err != nil {
		return eris.Wrapf(err, "Failed to remove network %s", cfg.DockerEnv["CARDINAL_NAMESPACE"])
	}

	return nil
}

func Purge(cfg *config.Config) error {
	cli, err := getCli()
	if err != nil {
		return eris.Wrap(err, "Failed to create docker client")
	}
	defer func() {
		err := cli.Close()
		if err != nil {
			logger.Error("Failed to close docker client", err)
		}
	}()

	err = Stop(cfg, Nakama, NakamaDB, Cardinal, Redis, CelestiaDevNet, EVM)
	if err != nil {
		return err
	}

	err = removeVolume(context.Background(), cli, cfg.DockerEnv["CARDINAL_NAMESPACE"])
	if err != nil {
		return err
	}

	return nil
}

func Restart(cfg *config.Config, configurations ...GetDockerConfig) error { //nolint:gocognit
	cli, err := getCli()
	if err != nil {
		return eris.Wrap(err, "Failed to create docker client")
	}
	defer func() {
		if !cfg.Detach {
			err := Stop(cfg, configurations...)
			if err != nil {
				logger.Error("Failed to stop containers", err)
			}
		}

		err := cli.Close()
		if err != nil {
			logger.Error("Failed to close docker client", err)
		}
	}()

	ctx := context.Background()
	for _, c := range configurations {
		configuration := c(cfg)
		if err := stopAndRemoveContainer(ctx, cli, configuration.Name); err != nil {
			return eris.Wrap(err, "Failed to stop container")
		}
	}

	// var for storing container names
	containers := make([]string, 0)

	// iterate over configurations and create containers
	for _, c := range configurations {
		configuration := c(cfg)
		if configuration.Dockerfile == nil {
			if err := pullImageIfNotExists(ctx, cli, configuration.Image); err != nil {
				return eris.Wrap(err, "Failed to pull image")
			}
		} else if cfg.Build {
			if err := buildImage(ctx, cli, *configuration.Dockerfile, configuration.Image); err != nil {
				return eris.Wrap(err, "Failed to build image")
			}
		}

		if err := createContainer(ctx, cli, configuration); err != nil {
			return eris.Wrap(err, "Failed to create container")
		}

		containers = append(containers, configuration.Name)
	}

	// log containers if not detached
	if !cfg.Detach {
		logContainers(cli, containers)
	}

	return nil
}

func getCli() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func createNetworkIfNotExists(cli *client.Client, networkName string) error {
	ctx := context.Background()
	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return err
	}

	for _, network := range networks {
		if network.Name == networkName {
			logger.Infof("Network %s already exists", networkName)
			return nil
		}
	}

	_, err = cli.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
	})
	if err != nil {
		return err
	}

	return nil
}

func createVolumeIfNotExists(cli *client.Client, volumeName string) error {
	ctx := context.Background()
	volumes, err := cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return err
	}

	for _, volume := range volumes.Volumes {
		if volume.Name == volumeName {
			logger.Debugf("Volume %s already exists\n", volumeName)
			return nil
		}
	}

	_, err = cli.VolumeCreate(ctx, volume.CreateOptions{Name: volumeName})
	if err != nil {
		return err
	}

	fmt.Printf("Created volume %s\n", volumeName)
	return nil
}

func createContainer(ctx context.Context, cli *client.Client, configuration Config) error {
	resp, err := cli.ContainerCreate(ctx, configuration.Config, configuration.HostConfig,
		configuration.NetworkingConfig, configuration.Platform, configuration.Name)
	if err != nil {
		return err
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	return nil
}

func pullImageIfNotExists(ctx context.Context, cli *client.Client, imageName string) error {
	_, _, err := cli.ImageInspectWithRaw(ctx, imageName)

	// If image exists, return
	if err == nil {
		logger.Println("Image already exists", imageName)
		return nil
	}

	// If image does not exist, pull it
	if client.IsErrNotFound(err) {
		out, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(os.Stdout, out)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

func buildImage(ctx context.Context, cli *client.Client, dockerfile Dockerfile, imageName string) error {
	fmt.Println("Building image ", imageName)
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Add the Dockerfile to the tar archive
	header := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfile.Script)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tw.Write([]byte(dockerfile.Script)); err != nil {
		return err
	}

	// Add source code to the tar archive
	if err := addFileToTarWriter(".", tw); err != nil {
		return err
	}

	// Read the tar archive
	tarReader := bytes.NewReader(buf.Bytes())

	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{imageName},
		Target:     dockerfile.Target,
		Version:    types.BuilderBuildKit,
	}

	// Build the image
	buildResponse, err := cli.ImageBuild(ctx, tarReader, buildOptions)
	if err != nil {
		return err
	}
	defer buildResponse.Body.Close()

	// Print the build logs
	if logger.VerboseMode {
		_, err = io.Copy(os.Stdout, buildResponse.Body)
	}

	fmt.Println("Image built successfully")
	return err
}

func stopAndRemoveContainer(ctx context.Context, cli *client.Client, containerName string) error {
	fmt.Printf("Removing %s...", containerName)

	// Check if the container exists
	_, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			fmt.Println(" Done")
			return nil // or return an error if you prefer
		}
		return eris.Wrapf(err, "Failed to inspect container %s", containerName)
	}

	// Stop the container
	err = cli.ContainerStop(ctx, containerName, container.StopOptions{})
	if err != nil {
		logger.Println("Failed to stop container", err)
		return eris.Wrapf(err, "Failed to stop container %s", containerName)
	}

	// Remove the container
	err = cli.ContainerRemove(ctx, containerName, container.RemoveOptions{})
	if err != nil {
		return eris.Wrapf(err, "Failed to remove container %s", containerName)
	}

	fmt.Println(" Done")

	return nil
}

func removeNetwork(ctx context.Context, cli *client.Client, networkName string) error {
	fmt.Printf("Removing network %s...", networkName)

	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return err
	}

	networkExist := false
	for _, network := range networks {
		if network.Name == networkName {
			networkExist = true
			break
		}
	}

	if networkExist {
		err = cli.NetworkRemove(ctx, networkName)
		if err != nil {
			return err
		}
	}

	fmt.Println(" Done")
	return nil
}

func removeVolume(ctx context.Context, cli *client.Client, volumeName string) error {
	fmt.Printf("Removing volume %s...", volumeName)

	err := cli.VolumeRemove(ctx, volumeName, true)
	if err != nil {
		return eris.Wrapf(err, "Failed to remove volume %s", volumeName)
	}

	fmt.Println(" Done")
	return nil
}

func logContainers(cli *client.Client, containers []string) {
	logs := make(map[string]io.ReadCloser)
	for _, c := range containers {
		out, err := cli.ContainerLogs(context.Background(), c, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
		if err != nil {
			panic(err)
		}
		logs[c] = out
		defer out.Close()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	for name, log := range logs {
		go func(name string, log io.ReadCloser) {
			buf := make([]byte, 4096) //nolint:gomnd
			for {
				n, err := log.Read(buf)
				if n > 0 {
					fmt.Printf("[%s] %s", name, buf[:n])
				}
				if err != nil {
					break
				}
			}
		}(name, log)
	}

	<-stop
}

// AddFileToTarWriter adds a file or directory to the tar writer
func addFileToTarWriter(baseDir string, tw *tar.Writer) error {
	return filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a tar header for the file or directory
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Adjust the header name to be relative to the baseDir
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		// Write the header to the tar writer
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// If it's a directory, there's no need to write file content
		if info.IsDir() {
			return nil
		}

		// Write the file content to the tar writer
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(tw, file); err != nil {
			return err
		}

		return nil
	})
}
