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
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/jsonmessage"
	controlapi "github.com/moby/buildkit/api/services/control"
	"github.com/rotisserie/eris"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"pkg.world.dev/world-cli/cmd/world/forge"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/tea/component/multispinner"
	"pkg.world.dev/world-cli/tea/style"
)

func (c *Client) buildImages(ctx context.Context, dockerServices ...service.Service) error {
	// Filter all services that need to be built
	var (
		serviceToBuild []service.Service
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

	p := forge.NewTeaProgram(multispinner.CreateSpinner(imagesName, cancel))

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

func (c *Client) buildImage(ctx context.Context, dockerService service.Service) (*types.ImageBuildResponse, error) {
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

	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{dockerService.Image},
		Target:     dockerService.BuildTarget,
	}

	if service.BuildkitSupport {
		buildOptions.Version = types.BuilderBuildKit
	}

	// Build the image
	buildResponse, err := c.client.ImageBuild(ctx, tarReader, buildOptions)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to build image")
	}

	return &buildResponse, nil
}

// The tar file is used to build the Docker image.
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
		if info.Name() != "world.toml" && !strings.HasPrefix(filepath.ToSlash(relPath), "cardinal/") {
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
			if service.BuildkitSupport {
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
	} else if err != nil {
		return "", eris.Wrap(err, "Error decoding build output")
	}

	// Only show the step if it's a build step
	step := ""
	if val, ok := event["stream"]; ok && val != "" && strings.HasPrefix(val.(string), "Step") {
		if step, ok = val.(string); ok && step != "" {
			step = strings.TrimSpace(step)
		}
	} else if val, ok = event["error"]; ok && val != "" {
		return "", eris.New(val.(string))
	}

	return step, nil
}

// filterImages filters the images that need to be pulled.
// Remove duplicates.
// Remove images that are already pulled.
// Remove images that need to be built.
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
			if service.OS != "" {
				images[service.Image] = fmt.Sprintf("%s/%s", service.OS, service.Architecture)
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

// Pulls the image if it does not exist.
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

		// Create a new progress bar for this image
		bar := p.AddBar(100, //nolint:gomnd
			mpb.PrependDecorators(
				decor.Name(fmt.Sprintf("%s %s: ", style.ForegroundPrint("Pulling", "2"), imageName)),
				decor.Percentage(decor.WCSyncSpace),
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
				printer.Infof("Error pulling image %s: %v\n", imageName, err)
				errChan <- eris.Wrapf(err, "error pulling image %s", imageName)

				// Stop the progress bar without clearing
				bar.Abort(false)
				return
			}
			defer responseBody.Close()

			// Process each event and update the progress bar
			decoder := json.NewDecoder(responseBody)
			var current int
			var event map[string]interface{}
			for decoder.More() { //nolint:dupl // different commands
				select {
				case <-ctx.Done():
					// Handle context cancellation
					printer.Infof("Pulling of image %s was canceled\n", imageName)
					bar.Abort(false) // Stop the progress bar without clearing
					return
				default:
					if err := decoder.Decode(&event); err != nil {
						errChan <- eris.New(fmt.Sprintf("Error decoding event for %s: %v\n", imageName, err))
						continue
					}

					// Check for errorDetail and error fields
					if errorDetail, ok := event["errorDetail"]; ok {
						if errorMessage, okay := errorDetail.(map[string]interface{})["message"]; okay {
							errChan <- eris.New(errorMessage.(string))
							continue
						}
					} else if errorMsg, okay := event["error"]; okay {
						errChan <- eris.New(errorMsg.(string))
						continue
					}

					// Handle progress updates
					if progressDetail, ok := event["progressDetail"].(map[string]interface{}); ok {
						if total, okay := progressDetail["total"].(float64); okay && total > 0 {
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

// Pulls the image if it does not exist.
func (c *Client) pushImages(ctx context.Context, pushTo string, authString string, //nolint:gocognit
	services ...service.Service) error {
	// Create a new progress container with a wait group
	var wg sync.WaitGroup
	p := mpb.New(mpb.WithWaitGroup(&wg))

	// Channel to collect errors from the goroutines
	errChan := make(chan error, len(services))

	// Add a wait group counter for each image
	wg.Add(len(services))

	for _, service := range services {
		imageName := service.Image

		// check if the image exists
		_, _, err := c.client.ImageInspectWithRaw(ctx, imageName)
		if err != nil {
			return eris.New(fmt.Sprintf("Error inspecting image %s for service %s: %v\n",
				imageName, service.Name, err))
		}

		bar := p.AddBar(100, //nolint:gomnd
			mpb.PrependDecorators(
				decor.Name(fmt.Sprintf("%s %s: ", style.ForegroundPrint("Pushing", "2"), imageName)),
				decor.Percentage(decor.WCSyncSpace),
			),
		)

		go func() {
			defer wg.Done()

			// Start pushing the image
			responseBody, err := c.client.ImagePush(ctx, pushTo, image.PushOptions{
				All:          true,
				RegistryAuth: authString,
			})

			if err != nil {
				// Handle the error: log it and send it to the error channel
				printer.Infof("Error pushing image %s: %v\n", imageName, err)
				errChan <- eris.Wrapf(err, "error pushing image %s", imageName)

				// Stop the progress bar without clearing
				bar.Abort(false)
				return
			}
			defer responseBody.Close()

			// Process each event and update the progress bar
			decoder := json.NewDecoder(responseBody)
			var current int
			var event map[string]interface{}
			for decoder.More() { //nolint:dupl // different commands
				select {
				case <-ctx.Done():
					// Handle context cancellation
					printer.Infof("Pushing image %s was canceled\n", imageName)
					bar.Abort(false) // Stop the progress bar without clearing
					return
				default:
					if err := decoder.Decode(&event); err != nil {
						errChan <- eris.New(fmt.Sprintf("Error decoding event for %s: %v\n", imageName, err))
						continue
					}

					// Check for errorDetail and error fields
					if errorDetail, ok := event["errorDetail"]; ok {
						if errorMessage, okay := errorDetail.(map[string]interface{})["message"]; okay {
							errChan <- eris.New(errorMessage.(string))
							continue
						}
					} else if errorMsg, okay := event["error"]; okay {
						errChan <- eris.New(errorMsg.(string))
						continue
					}

					// Handle progress updates
					if progressDetail, ok := event["progressDetail"].(map[string]interface{}); ok {
						if total, okay := progressDetail["total"].(float64); okay && total > 0 {
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
