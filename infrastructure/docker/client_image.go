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
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/jsonmessage"
	controlapi "github.com/moby/buildkit/api/services/control"
	"github.com/rotisserie/eris"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"pkg.world.dev/world-cli/infrastructure/docker/types"
	"pkg.world.dev/world-cli/ui/component/multispinner"
	"pkg.world.dev/world-cli/ui/style"
)

// PullEventHandler handles Docker pull events and updates progress
type PullEventHandler struct {
	bar       *mpb.Bar
	progress  *float64
	errChan   chan<- error
	imageName string
}

// NewPullEventHandler creates a new PullEventHandler instance
func NewPullEventHandler(bar *mpb.Bar, progress *float64, errChan chan<- error, imageName string) *PullEventHandler {
	return &PullEventHandler{
		bar:       bar,
		progress:  progress,
		errChan:   errChan,
		imageName: imageName,
	}
}

func (c *Client) buildImages(ctx context.Context, dockerServices ...types.Service) error {
	// Filter all services that need to be built
	var (
		serviceToBuild []types.Service
		imagesName     []string
	)
	for _, dockerService := range dockerServices {
		if dockerService.Dockerfile != "" {
			serviceToBuild = append(serviceToBuild, dockerService)
			imagesName = append(imagesName, dockerService.Image)
		}
	}
	if len(serviceToBuild) == 0 {
		return nil
	}

	// Create ctx with cancel
	ctx, cancel := context.WithCancel(ctx)

	// Channel to collect errors from the goroutines
	errChan := make(chan error, len(imagesName))

	p := tea.NewProgram(multispinner.CreateSpinner(imagesName, cancel))

	for _, ds := range serviceToBuild {
		// Capture dockerService in the loop
		dockerService := ds

		go func() {
			p.Send(multispinner.ProcessState{
				State: "building",
				Type:  "image",
				Name:  dockerService.Image,
			})

			// Remove the container
			err := c.removeContainer(ctx, dockerService.Name)
			if err != nil {
				p.Send(multispinner.ProcessState{
					Icon:   style.CrossIcon.Render(),
					State:  "building",
					Type:   "image",
					Name:   dockerService.Image,
					Detail: err.Error(),
					Done:   true,
				})
				errChan <- err
				return
			}

			// Start the build process
			buildResponse, err := c.buildImage(ctx, dockerService)
			if err != nil {
				p.Send(multispinner.ProcessState{
					Icon:   style.CrossIcon.Render(),
					State:  "building",
					Type:   "image",
					Name:   dockerService.Image,
					Detail: err.Error(),
					Done:   true,
				})
				errChan <- err
				return
			}
			defer buildResponse.Body.Close()

			// Print the build logs
			err = c.readBuildLog(ctx, buildResponse.Body, p, dockerService.Image)
			if err != nil {
				errChan <- err
			}
		}()
	}

	// Run the program
	if _, err := p.Run(); err != nil {
		return eris.Wrap(err, "Error running program")
	}

	// Close the error channel and check for errors
	close(errChan)
	errs := make([]error, 0)
	for err := range errChan {
		errs = append(errs, err)
	}

	// If there were any errors, return them as a combined error
	if len(errs) > 0 {
		return eris.New(fmt.Sprintf("Errors: %v", errs))
	}

	return nil
}

func (c *Client) buildImage(ctx context.Context, dockerService types.Service) (*dockertypes.ImageBuildResponse, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Add the Dockerfile to the tar archive
	header := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerService.Dockerfile)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return nil, eris.Wrap(err, "Failed to write header to tar writer")
	}
	if _, err := tw.Write([]byte(dockerService.Dockerfile)); err != nil {
		return nil, eris.Wrap(err, "Failed to write Dockerfile to tar writer")
	}

	// Add source code to the tar archive
	if err := c.addFileToTarWriter(c.cfg.RootDir, tw); err != nil {
		return nil, eris.Wrap(err, "Failed to add source code to tar writer")
	}

	// Read the tar archive
	tarReader := bytes.NewReader(buf.Bytes())

	buildOptions := dockertypes.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{dockerService.Image},
		Target:     dockerService.BuildTarget,
	}

	if types.BuildkitSupport {
		buildOptions.Version = dockertypes.BuilderBuildKit
	}

	// Build the image
	buildResponse, err := c.client.ImageBuild(ctx, tarReader, buildOptions)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to build image")
	}

	return &buildResponse, nil
}

// AddFileToTarWriter adds a file or directory to the tar writer
// This function is used to add the Dockerfile and source code to the tar archive
// The tar file is used to build the Docker image
func (c *Client) addFileToTarWriter(baseDir string, tw *tar.Writer) error {
	return filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return eris.Wrapf(err, "Failed to walk the directory %s", baseDir)
		}

		// Check if the file is world.toml or inside the cardinal directory
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return eris.Wrapf(err, "Failed to get relative path %s", path)
		}
		// Skip files that are not world.toml or inside the cardinal directory
		if !(info.Name() == "world.toml" || strings.HasPrefix(filepath.ToSlash(relPath), "cardinal/")) {
			return nil
		}

		// Create a tar header for the file or directory
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return eris.Wrap(err, "Failed to create tar header")
		}

		// Adjust the header name to be relative to the baseDir
		header.Name = filepath.ToSlash(relPath)

		// Write the header to the tar writer
		if err := tw.WriteHeader(header); err != nil {
			return eris.Wrap(err, "Failed to write header to tar writer")
		}

		// If it's a directory, there's no need to write file content
		if info.IsDir() {
			return nil
		}

		// Write the file content to the tar writer
		file, err := os.Open(path)
		if err != nil {
			return eris.Wrapf(err, "Failed to open file %s", path)
		}
		defer file.Close()

		if _, err := io.Copy(tw, file); err != nil {
			return eris.Wrap(err, "Failed to copy file to tar writer")
		}

		return nil
	})
}

// readBuildLog filters the output of the Docker build command
// there is two types of output:
// 1. Output from the build process without buildkit
//   - stream: Output from the build process
//   - error: Error message from the build process
//
// 2. Output from the build process with buildkit
//   - moby.buildkit.trace: Output from the build process
//   - error: Need to research how buildkit log send error messages (TODO)
func (c *Client) readBuildLog(ctx context.Context, reader io.Reader, p *tea.Program, imageName string) error {
	// Create a new JSON decoder
	decoder := json.NewDecoder(reader)

	for stop := false; !stop; {
		select {
		case <-ctx.Done():
			stop = true
		default:
			var step string
			var err error
			if types.BuildkitSupport {
				// Parse the buildkit response
				step, err = c.parseBuildkitResp(decoder, &stop)
			} else {
				// Parse the non-buildkit response
				step, err = c.parseNonBuildkitResp(decoder, &stop)
			}

			// Send the step to the spinner
			if err != nil {
				p.Send(multispinner.ProcessState{
					Icon:   style.CrossIcon.Render(),
					State:  "building",
					Type:   "image",
					Name:   imageName,
					Detail: err.Error(),
					Done:   true,
				})
				return err
			}

			if step != "" {
				p.Send(multispinner.ProcessState{
					State:  "building",
					Type:   "image",
					Name:   imageName,
					Detail: step,
				})
			}
		}
	}

	// Send the final message to the spinner
	p.Send(multispinner.ProcessState{
		Icon:  style.TickIcon.Render(),
		State: "built",
		Type:  "image",
		Name:  imageName,
		Done:  true,
	})

	return nil
}

func (c *Client) parseBuildkitResp(decoder *json.Decoder, stop *bool) (string, error) {
	var msg jsonmessage.JSONMessage
	if err := decoder.Decode(&msg); errors.Is(err, io.EOF) {
		*stop = true
	} else if err != nil {
		return "", eris.Wrap(err, "Error decoding build output")
	}

	var resp controlapi.StatusResponse

	if msg.ID != "moby.buildkit.trace" {
		return "", nil
	}

	var dt []byte
	// ignoring all messages that are not understood
	// need to research how buildkit log send error messages
	if err := json.Unmarshal(*msg.Aux, &dt); err != nil {
		return "", nil //nolint:nilerr // ignore error
	}
	if err := (&resp).Unmarshal(dt); err != nil {
		return "", nil //nolint:nilerr // ignore error
	}

	if len(resp.Vertexes) == 0 {
		return "", nil
	}

	// return the name of the vertex (step) that is currently being executed
	return resp.Vertexes[len(resp.Vertexes)-1].Name, nil
}

func (c *Client) parseNonBuildkitResp(decoder *json.Decoder, stop *bool) (string, error) {
	var event map[string]interface{}
	if err := decoder.Decode(&event); errors.Is(err, io.EOF) {
		*stop = true
		return "", nil
	} else if err != nil {
		return "", eris.Wrap(err, "Error decoding build output")
	}

	if step := c.extractStreamStep(event); step != "" {
		return step, nil
	}

	return c.checkForErrors(event)
}

// extractStreamStep extracts build step information from stream events
func (c *Client) extractStreamStep(event map[string]interface{}) string {
	val, ok := event["stream"].(string)
	if !ok || val == "" {
		return ""
	}
	if strings.HasPrefix(val, "Step") {
		return strings.TrimSpace(val)
	}
	return ""
}

// checkForErrors checks for error information in the event
func (c *Client) checkForErrors(event map[string]interface{}) (string, error) {
	if val, ok := event["error"].(string); ok && val != "" {
		return "", eris.New(val)
	}
	return "", nil
}

// filterImages filters the images that need to be pulled
// Remove duplicates
// Remove images that are already pulled
// Remove images that need to be built
func (c *Client) filterImages(ctx context.Context, images map[string]string, services ...types.Service) {
	for _, service := range services {
		// check if the image exists
		_, _, err := c.client.ImageInspectWithRaw(ctx, service.Image)
		if err == nil {
			// Image already exists, skip pulling
			continue
		}

		// check if the image needs to be built
		// if the service has a Dockerfile, it needs to be built
		if service.Dockerfile == "" {
			// Image does not exist and does not need to be built
			// Add the image to the list of images to pull
			if service.Platform.OS != "" {
				images[service.Image] = fmt.Sprintf("%s/%s", service.Platform.OS, service.Platform.Architecture)
			} else {
				images[service.Image] = ""
			}
		}

		// Recursively check dependencies
		if service.Dependencies != nil {
			c.filterImages(ctx, images, service.Dependencies...)
		}
	}
}

// Pulls the image if it does not exist
func (c *Client) pullImages(ctx context.Context, services ...types.Service) error {
	// Filter the images that need to be pulled
	images := make(map[string]string)
	c.filterImages(ctx, images, services...)

	// Create a new progress container with a wait group
	var wg sync.WaitGroup
	p := mpb.New(mpb.WithWaitGroup(&wg))

	// Channel to collect errors from the goroutines
	errChan := make(chan error, len(images))

	// Add a wait group counter for each image
	wg.Add(len(images))

	// Pull each image concurrently
	for imageName, platform := range images {
		// Create a new progress bar for this image
		bar := p.AddBar(types.ProgressBarMax,
			mpb.PrependDecorators(
				decor.Name(fmt.Sprintf("%s %s: ", style.ForegroundPrint("Pulling", "2"), imageName)),
				decor.Percentage(decor.WCSyncSpace),
			),
		)

		go func(imageName, platform string) {
			defer wg.Done()
			c.handleImagePull(ctx, imageName, platform, bar, errChan)
		}(imageName, platform)
	}

	// Wait for all progress bars to complete
	wg.Wait()
	p.Wait()

	// Close the error channel and check for errors
	close(errChan)
	errs := make([]error, 0)
	for err := range errChan {
		errs = append(errs, err)
	}

	// If there were any errors, return them as a combined error
	if len(errs) > 0 {
		return eris.New(fmt.Sprintf("Errors: %v", errs))
	}

	return nil
}

// handleImagePull handles the pulling of a single image and updates its progress bar
func (c *Client) handleImagePull(ctx context.Context, imageName, platform string, bar *mpb.Bar, errChan chan<- error) {
	// Start pulling the image
	responseBody, err := c.client.ImagePull(ctx, imageName, image.PullOptions{
		Platform: platform,
	})

	if err != nil {
		fmt.Printf("Error pulling image %s: %v\n", imageName, err)
		errChan <- eris.Wrapf(err, "error pulling image %s", imageName)
		bar.Abort(false)
		return
	}
	defer responseBody.Close()

	// Process each event and update the progress bar
	decoder := json.NewDecoder(responseBody)
	var progress float64

	for decoder.More() {
		select {
		case <-ctx.Done():
			fmt.Printf("Pulling of image %s was canceled\n", imageName)
			bar.Abort(false)
			return
		default:
			if err := c.handlePullEvent(decoder, bar, &progress, errChan, imageName); err != nil {
				return
			}
		}
	}

	// Complete the progress bar
	bar.SetCurrent(types.ProgressBarMax)
}

// handlePullEvent processes a single Docker pull event and updates progress
func (c *Client) handlePullEvent(
	decoder *json.Decoder,
	bar *mpb.Bar,
	progress *float64,
	errChan chan<- error,
	imageName string,
) error {
	handler := NewPullEventHandler(bar, progress, errChan, imageName)
	return handler.processEvent(decoder)
}

// processEvent decodes and processes a single Docker pull event
func (h *PullEventHandler) processEvent(decoder *json.Decoder) error {
	var event map[string]interface{}
	if err := decoder.Decode(&event); err != nil {
		msg := fmt.Sprintf("Error decoding event for %s: %v", h.imageName, err)
		h.errChan <- eris.New(msg)
		return nil
	}

	if err := h.handleError(event); err != nil {
		return err
	}

	h.updateProgress(event)
	return nil
}

// updateProgress updates the progress bar based on event data
func (h *PullEventHandler) updateProgress(event map[string]interface{}) {
	progressDetail, ok := event["progressDetail"].(map[string]interface{})
	if !ok || progressDetail == nil {
		return
	}

	current, ok := progressDetail["current"].(float64)
	if !ok {
		return
	}

	total, ok := progressDetail["total"].(float64)
	if !ok || total <= 0 {
		return
	}

	calculatedProgress := current * float64(types.ProgressBarMax) / total
	if calculatedProgress > *h.progress {
		h.bar.SetCurrent(int64(calculatedProgress))
		*h.progress = calculatedProgress
	}
}

// handleError processes error information from Docker pull events
func (h *PullEventHandler) handleError(event map[string]interface{}) error {
	if err := h.processErrorDetail(event); err != nil {
		return err
	}
	return h.processErrorMessage(event)
}

// processErrorDetail handles detailed error information
func (h *PullEventHandler) processErrorDetail(event map[string]interface{}) error {
	errorDetail, ok := event["errorDetail"].(map[string]interface{})
	if !ok || errorDetail == nil {
		return nil
	}

	msg, ok := errorDetail["message"].(string)
	if !ok {
		h.errChan <- eris.New("unknown error format in pull output")
		return eris.New("unknown error format in pull output")
	}

	h.errChan <- eris.New(msg)
	return eris.New(msg)
}

// processErrorMessage handles simple error messages
func (h *PullEventHandler) processErrorMessage(event map[string]interface{}) error {
	errorMsg, ok := event["error"].(string)
	if !ok || errorMsg == "" {
		return nil
	}

	h.errChan <- eris.New(errorMsg)
	return eris.New(errorMsg)
}
