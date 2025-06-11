package forge

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
)

// deploymentBuild builds the image and returns the commit hash and the image reader.
func deploymentBuild(ctx context.Context, project *project) (string, io.ReadCloser, error) {
	tempDir, err := os.MkdirTemp("", "wfbuild")
	if err != nil {
		return "", nil, eris.Wrapf(err, "Failed to create temp dir")
	}

	commitHash, err := cloneRepo(project.RepoURL, project.RepoToken, tempDir)
	if err != nil {
		return "", nil, eris.Wrapf(err, "Failed to clone repo")
	}

	// build the image
	reader, err := buildImage(ctx, project.RepoPath, tempDir)
	if err != nil {
		return "", nil, eris.Wrapf(err, "Failed to build image")
	}

	return commitHash, reader, nil
}

func cloneRepo(repoURL, token string, tempDir string) (string, error) {
	env := map[string]string{
		"GIT_COMMITTER_NAME":  "World CLI",
		"GIT_COMMITTER_EMAIL": "no-reply@world.dev",
	}

	outBuff := bytes.Buffer{}
	errBuff := bytes.Buffer{}

	// shallow clone the repo
	if token != "" {
		// Add token to the URL for authentication
		repoURLWithToken := strings.Replace(repoURL, "https://", "https://"+token+"@", 1)
		repoURL = repoURLWithToken
	}
	_, err := sh.Exec(env, &outBuff, &errBuff, "git", "clone", "--depth", "1", repoURL, tempDir)
	if err != nil {
		return "", eris.Wrapf(err, "failed to clone repo: %s", errBuff.String())
	}

	// get the commit hash
	_, err = sh.Exec(env, &outBuff, &errBuff, "git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", eris.Wrapf(err, "failed to get commit hash: %s", errBuff.String())
	}

	commitHash := outBuff.String()

	return commitHash, nil
}

func buildImage(ctx context.Context, repoPath, tempDir string) (io.ReadCloser, error) {
	// get config from world.toml
	worldTomlPath := path.Join(tempDir, repoPath, "world.toml")
	cfg, err := config.GetConfig(&worldTomlPath)
	if err != nil {
		return nil, eris.Wrapf(err, "Failed to get config")
	}

	// create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return nil, eris.Wrapf(err, "Failed to create docker client")
	}

	// build the image
	err = dockerClient.Build(ctx, "", "", service.Cardinal)
	if err != nil {
		return nil, eris.Wrapf(err, "Failed to build image")
	}

	// get the image name
	cardinalService := service.Cardinal(cfg)
	imageName := cardinalService.Image

	// save the image
	reader, err := dockerClient.Save(ctx, imageName)
	if err != nil {
		return nil, eris.Wrapf(err, "Failed to save image")
	}

	return reader, nil
}
