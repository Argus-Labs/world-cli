package forge

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/magefile/mage/sh"
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
)

type temporaryCredential struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
	Region          string `json:"region"`
	RepoURI         string `json:"repo_uri"`
}

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

func pushImage(fCtx ForgeContext, tag string, buf bytes.Buffer) error {
	// get temporary credentials
	getTempCredURL := fmt.Sprintf("%s/api/organization/%s/project/%s/temporary-credential",
		baseURL, fCtx.State.Organization.ID, fCtx.State.Project.ID)
	result, err := sendRequest(fCtx, http.MethodGet, getTempCredURL, nil)
	if err != nil {
		return eris.Wrapf(err, "Failed to get temporary credentials")
	}

	// parse the result
	tempCred, err := parseResponse[temporaryCredential](result)
	if err != nil {
		return eris.Wrapf(err, "Failed to parse temporary credentials")
	}

	// push the image to the registry
	ref, err := name.ParseReference(tempCred.RepoURI + ":" + tag)
	if err != nil {
		return eris.Wrapf(err, "Failed to parse reference")
	}

	// Create a custom opener that uses the buffered data
	opener := func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
	}

	// read the image from the tarball
	img, err := tarball.Image(opener, nil)
	if err != nil {
		return eris.Wrapf(err, "Failed to read tarball")
	}

	// get ECR auth
	auth, err := getECRAuth(*tempCred)
	if err != nil {
		return eris.Wrapf(err, "Failed to get ECR auth")
	}

	// push the image to the registry
	err = remote.Write(ref, img, remote.WithAuth(auth))
	if err != nil {
		return eris.Wrapf(err, "Failed to push image")
	}

	return nil
}

func getECRAuth(tempCred temporaryCredential) (authn.Authenticator, error) {
	// load AWS config
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, eris.Wrapf(err, "Failed to load AWS config")
	}
	cfg.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		tempCred.AccessKeyID,
		tempCred.SecretAccessKey,
		tempCred.SessionToken,
	))
	// set region
	cfg.Region = tempCred.Region
	ecrClient := ecr.NewFromConfig(cfg)
	resp, err := ecrClient.GetAuthorizationToken(context.TODO(), &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return nil, eris.Wrapf(err, "Failed to get authorization token")
	}
	token := *resp.AuthorizationData[0].AuthorizationToken
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, eris.Wrapf(err, "Failed to decode authorization token")
	}
	parts := strings.SplitN(string(decoded), ":", 2)

	return &authn.Basic{Username: parts[0], Password: parts[1]}, nil
}
