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

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/jsonmessage"
	controlapi "github.com/moby/buildkit/api/services/control"
	"github.com/rotisserie/eris"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/logger"
	teaspinner "pkg.world.dev/world-cli/tea/component/spinner"
)

func (c *Client) buildImage(ctx context.Context, dockerfile, target, imageName string) error {
	contextPrint("Building", "2", "image", imageName)
	fmt.Println() // Add a newline after the context print
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Add the Dockerfile to the tar archive
	header := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfile)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return eris.Wrap(err, "Failed to write header to tar writer")
	}
	if _, err := tw.Write([]byte(dockerfile)); err != nil {
		return eris.Wrap(err, "Failed to write Dockerfile to tar writer")
	}

	// Add source code to the tar archive
	if err := c.addFileToTarWriter(".", tw); err != nil {
		return eris.Wrap(err, "Failed to add source code to tar writer")
	}

	// Read the tar archive
	tarReader := bytes.NewReader(buf.Bytes())

	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{imageName},
		Target:     target,
	}

	if service.BuildkitSupport {
		buildOptions.Version = types.BuilderBuildKit
	}

	// Build the image
	buildResponse, err := c.client.ImageBuild(ctx, tarReader, buildOptions)
	if err != nil {
		return err
	}
	defer buildResponse.Body.Close()

	// Print the build logs
	c.readBuildLog(ctx, buildResponse.Body)

	return nil
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
func (c *Client) readBuildLog(ctx context.Context, reader io.Reader) {
	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)

	// Create a new JSON decoder
	decoder := json.NewDecoder(reader)

	// Initialize the spinner
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	s.Spinner = spinner.Points

	// Initialize the model
	m := teaspinner.Spinner{
		Spinner: s,
		Cancel:  cancel,
	}

	// Start the bubbletea program
	p := tea.NewProgram(m)
	go func() {
		for stop := false; !stop; {
			select {
			case <-ctx.Done():
				stop = true
			default:
				var step string
				if service.BuildkitSupport {
					// Parse the buildkit response
					step = c.parseBuildkitResp(decoder, &stop)
				} else {
					// Parse the non-buildkit response
					step = c.parseNonBuildkitResp(decoder, &stop)
				}

				// Send the step to the spinner
				if step != "" {
					p.Send(teaspinner.LogMsg(step))
				}
			}
		}
		// Send a completion message to the spinner
		p.Send(teaspinner.LogMsg("spin: completed"))
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
	}
}

func (c *Client) parseBuildkitResp(decoder *json.Decoder, stop *bool) string {
	var msg jsonmessage.JSONMessage
	if err := decoder.Decode(&msg); errors.Is(err, io.EOF) {
		*stop = true
	} else if err != nil {
		logger.Errorf("Error decoding build output: %v", err)
	}

	var resp controlapi.StatusResponse

	if msg.ID != "moby.buildkit.trace" {
		return ""
	}

	var dt []byte
	// ignoring all messages that are not understood
	if err := json.Unmarshal(*msg.Aux, &dt); err != nil {
		return ""
	}
	if err := (&resp).Unmarshal(dt); err != nil {
		return ""
	}

	if len(resp.Vertexes) == 0 {
		return ""
	}

	// return the name of the vertex (step) that is currently being executed
	return resp.Vertexes[len(resp.Vertexes)-1].Name
}

func (c *Client) parseNonBuildkitResp(decoder *json.Decoder, stop *bool) string {
	var event map[string]interface{}
	if err := decoder.Decode(&event); errors.Is(err, io.EOF) {
		*stop = true
	} else if err != nil {
		logger.Errorf("Error decoding build output: %v", err)
	}

	// Only show the step if it's a build step
	step := ""
	if val, ok := event["stream"]; ok && val != "" && strings.HasPrefix(val.(string), "Step") {
		if step, ok = val.(string); ok && step != "" {
			step = strings.TrimSpace(step)
		}
	} else if val, ok = event["error"]; ok && val != "" {
		logger.Errorf("Error building image: %v", val)
	}

	return step
}

// filterImages filters the images that need to be pulled
// Remove duplicates
// Remove images that are already pulled
// Remove images that need to be built
func (c *Client) filterImages(ctx context.Context, images map[string]string, services ...service.Service) {
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
func (c *Client) pullImages(ctx context.Context, services ...service.Service) error { //nolint:gocognit
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
		// Capture imageName and platform in the loop
		imageName := imageName
		platform := platform

		// Create a new progress bar for this image
		bar := p.AddBar(100, //nolint:gomnd
			mpb.PrependDecorators(
				decor.Name(fmt.Sprintf("%s %s: ", foregroundPrint("Pulling", "2"), imageName)),
				decor.Percentage(decor.WCSyncSpace),
			),
			mpb.AppendDecorators(
				decor.OnComplete(
					decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WCSyncWidth), "done", //nolint:gomnd
				),
			),
		)

		go func() {
			defer wg.Done()

			// Start pulling the image
			responseBody, err := c.client.ImagePull(ctx, imageName, image.PullOptions{
				Platform: platform,
			})

			if err != nil {
				// Handle the error: log it and send it to the error channel
				fmt.Printf("Error pulling image %s: %v\n", imageName, err)
				errChan <- fmt.Errorf("error pulling image %s: %w", imageName, err)

				// Stop the progress bar without clearing
				bar.Abort(false)
				return
			}
			defer responseBody.Close()

			// Process each event and update the progress bar
			decoder := json.NewDecoder(responseBody)
			var current int
			var event map[string]interface{}
			for decoder.More() {
				select {
				case <-ctx.Done():
					// Handle context cancellation
					fmt.Printf("Pulling of image %s was canceled\n", imageName)
					bar.Abort(false) // Stop the progress bar without clearing
					return
				default:
					if err := decoder.Decode(&event); err != nil {
						errChan <- eris.New(fmt.Sprintf("Error decoding event for %s: %v\n", imageName, err))
						continue
					}

					// Check for errorDetail and error fields
					if errorDetail, ok := event["errorDetail"]; ok {
						if errorMessage, ok := errorDetail.(map[string]interface{})["message"]; ok {
							errChan <- eris.New(errorMessage.(string))
							continue
						}
					} else if errorMsg, ok := event["error"]; ok {
						errChan <- eris.New(errorMsg.(string))
						continue
					}

					// Handle progress updates
					if progressDetail, ok := event["progressDetail"].(map[string]interface{}); ok {
						if total, ok := progressDetail["total"].(float64); ok && total > 0 {
							calculatedCurrent := int(progressDetail["current"].(float64) * 100 / total)
							if calculatedCurrent > current {
								bar.SetCurrent(int64(calculatedCurrent))
								current = calculatedCurrent
							}
						}
					}
				}
			}

			// Finish the progress bar
			// Handle if the current and total is not available in the response body
			// Usually, because docker image is already pulled from the cache
			bar.SetCurrent(100) //nolint:gomnd
		}()
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
