package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/tea/style"
)

type BuildOutput struct {
	Stream string `json:"stream"`
	Aux    struct {
		ID string `json:"ID"`
	} `json:"aux"`
}

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
			if err := pullImageIfNotExists(ctx, cli, configuration); err != nil {
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

	return nil
}

func Purge(cfg *config.Config) error {
	ctx := context.Background()

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

	err = removeVolume(ctx, cli, cfg.DockerEnv["CARDINAL_NAMESPACE"])
	if err != nil {
		return err
	}

	err = removeNetwork(ctx, cli, cfg.DockerEnv["CARDINAL_NAMESPACE"])
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
			if err := pullImageIfNotExists(ctx, cli, configuration); err != nil {
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

func pullImageIfNotExists(ctx context.Context, cli *client.Client, configuration Config) error {
	fmt.Println("Pulling image", configuration.Image)

	_, _, err := cli.ImageInspectWithRaw(ctx, configuration.Image)

	// If image exists, return
	if err == nil {
		logger.Println("Image already exists", configuration.Image)
		return nil
	}

	// If image does not exist, pull it
	if client.IsErrNotFound(err) {
		pullOptions := image.PullOptions{}

		// Set platform if specified
		if configuration.Platform != nil {
			pullOptions.Platform = fmt.Sprintf("%s/%s", configuration.Platform.OS, configuration.Platform.Architecture)
		}

		out, err := cli.ImagePull(ctx, configuration.Image, pullOptions)
		if err != nil {
			return err
		}
		defer out.Close()

		return filterDockerPullOutput(out)
	}

	return err
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
	}

	if buildKitSupport {
		buildOptions.Version = types.BuilderBuildKit
	}

	// Build the image
	buildResponse, err := cli.ImageBuild(ctx, tarReader, buildOptions)
	if err != nil {
		return err
	}
	defer buildResponse.Body.Close()

	// Print the build logs
	err = filterDockerBuildOutput(buildResponse.Body)
	if err != nil {
		return err
	}

	return nil
}

func stopAndRemoveContainer(ctx context.Context, cli *client.Client, containerName string) error {
	text := contextPrint("Removing", "1", "container", "4", containerName)
	fmt.Print(text)

	// Check if the container exists
	_, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			fmt.Printf("\r%s %s\n", text, style.TickIcon.Render())
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

	fmt.Printf("\r%s %s\n", text, style.TickIcon.Render())
	return nil
}

func removeNetwork(ctx context.Context, cli *client.Client, networkName string) error {
	text := contextPrint("Removing", "1", "network", "4", networkName)
	fmt.Print(text)

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

	fmt.Printf("\r%s %s\n", text, style.TickIcon.Render())

	return nil
}

func removeVolume(ctx context.Context, cli *client.Client, volumeName string) error {
	text := contextPrint("Removing", "1", "volume", "4", volumeName)
	fmt.Print(text)

	err := cli.VolumeRemove(ctx, volumeName, true)
	if err != nil {
		return eris.Wrapf(err, "Failed to remove volume %s", volumeName)
	}

	fmt.Printf("\r%s %s\n", text, style.TickIcon.Render())
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

		// Check if the file is world.toml or inside the cardinal directory
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		if !(info.Name() == "world.toml" || strings.HasPrefix(filepath.ToSlash(relPath), "cardinal/")) {
			return nil
		}

		// Create a tar header for the file or directory
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Adjust the header name to be relative to the baseDir
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

func filterDockerBuildOutput(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	for {
		var event map[string]interface{}
		if err := decoder.Decode(&event); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}

		// Print only the 'stream' content
		if val, ok := event["Stream"]; ok && val != "" {
			fmt.Print(val)
		}
	}
	return nil
}

func filterDockerPullOutput(reader io.Reader) error { //nolint:gocognit
	decoder := json.NewDecoder(reader)
	var statusTemp interface{}
	for {
		var event map[string]interface{}
		if err := decoder.Decode(&event); errors.Is(err, io.EOF) {
			fmt.Println()
			break
		} else if err != nil {
			return err
		}

		// Check for errorDetail and error fields
		if errorDetail, ok := event["errorDetail"]; ok {
			if errorMessage, ok := errorDetail.(map[string]interface{})["message"]; ok {
				fmt.Printf("\r%s %s\n", foregroundPrint("Error:", "1"), errorMessage)
				break
			}
		} else if errorMsg, ok := event["error"]; ok {
			fmt.Printf("\r%s %s\n", foregroundPrint("Error:", "1"), errorMsg)
			break
		}

		// Filter and print relevant information
		if status, ok := event["status"]; ok {
			output, ok := status.(string)
			if !ok {
				logger.Errorf("Failed to cast status to string %v", status)
				continue
			}
			output = foregroundPrint(output, "4")
			if progress, ok := event["progress"]; ok {
				output += " " + progress.(string)
			}
			if statusTemp == status {
				fmt.Printf("\r%s", output)
			} else {
				fmt.Printf("\n%s", output)
			}
			statusTemp = status

			os.Stdout.Sync()
		}
	}
	return nil
}

func contextPrint(title string, titleColor string, subject string, subjectColor string, object string) string {
	titleStr := foregroundPrint(title, titleColor)
	arrowStr := foregroundPrint("→", "241")
	subjectStr := foregroundPrint(subject, subjectColor)

	return fmt.Sprintf("%s %s %s %s", titleStr, arrowStr, subjectStr, object)
}

func foregroundPrint(text string, color string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(text)
}
