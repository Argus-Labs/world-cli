package forge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rotisserie/eris"
)

// identifyProvider determines the Git provider based on the URL's host.
var identifyProvider = func(repoURL string) (string, string, error) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	host := parsedURL.Host
	switch {
	case strings.Contains(host, "github.com"):
		return "GitHub", "https://api.github.com", nil //nolint:goconst // test, don't care about constant
	case strings.Contains(host, "gitlab.com"):
		return "GitLab", "https://gitlab.com/api/v4", nil //nolint:goconst // test, don't care about constant
	case strings.Contains(host, "bitbucket.org"):
		return "Bitbucket", "https://api.bitbucket.org/2.0", nil //nolint:goconst // test, don't care about constant
	default:
		return "Unknown", "", fmt.Errorf("unknown provider: %s", host)
	}
}

// validateRepoToken tests if the token and repo URL are valid using the provider's API.
func validateRepoToken(ctx context.Context, repoURL, token string) error {
	provider, apiBaseURL, err := identifyProvider(repoURL)
	if err != nil {
		return fmt.Errorf("failed to identify provider: %w", err)
	}

	switch provider {
	case "GitHub":
		return validateGitHub(ctx, repoURL, token, apiBaseURL)
	case "GitLab":
		return validateGitLab(ctx, repoURL, token, apiBaseURL)
	case "Bitbucket":
		return validateBitbucket(ctx, repoURL, token, apiBaseURL)
	default:
		return fmt.Errorf("provider %s is not supported", provider)
	}
}

// params: ctx, repoURL, token, path
func validateRepoPath(_ context.Context, _, _, path string) error {
	if strings.Contains(path, " ") {
		return fmt.Errorf("invalid path: %s", path)
	}
	// I don't think we need to verify that the path actually exists in the repo,
	// but if we decide to here's where we would do that. If it doesn't exist then
	// any deploy attempt will fail in the World Forge Worker at the checkout action
	// Hints at possible GitHub implementation here: https://github.com/orgs/community/discussions/68413
	return nil
}

// validateGitHub validates the repository for GitHub, first trying public access.
func validateGitHub(ctx context.Context, repoURL, token, apiBaseURL string) error {
	// Extract the owner and repo name from the URL
	parts := strings.Split(repoURL, "/")
	if len(parts) < 2 { //nolint:gomnd
		return eris.New("invalid github repository URL")
	}
	repo := strings.TrimSuffix(parts[len(parts)-1], ".git")
	owner := parts[len(parts)-2]

	// Construct the API request URL
	apiURL := fmt.Sprintf("%s/repos/%s/%s", apiBaseURL, owner, repo)

	// First try accessing the repo without a token (public access)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If public access works, return success
	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ GitHub repository is public and accessible!")
		return nil
	}

	// If public access fails and no token provided, return error
	if token == "" {
		return errors.New("repository is not publicly accessible, please provide an access token")
	}

	// Try again with the token
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ GitHub repository access validated with token!")
		return nil
	}
	return fmt.Errorf("GitHub validation failed: %s", resp.Status)
}

// validateGitLab validates the repository for GitLab, first trying public access.
func validateGitLab(ctx context.Context, repoURL, token, apiBaseURL string) error {
	// Extract the project path from the URL
	parts := strings.Split(repoURL, "/")
	if len(parts) < 2 { //nolint:gomnd
		return eris.New("invalid gitlab repository URL")
	}
	projectPath := fmt.Sprintf("%s/%s", parts[len(parts)-2], strings.TrimSuffix(parts[len(parts)-1], ".git"))

	// Construct the API request URL
	apiURL := fmt.Sprintf("%s/projects/%s", apiBaseURL, url.QueryEscape(projectPath))

	// First try accessing the repo without a token (public access)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If public access works, return success
	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ GitLab repository is public and accessible!")
		return nil
	}

	// If public access fails and no token provided, return error
	if token == "" {
		return errors.New("repository is not publicly accessible, please provide an access token")
	}

	// Try again with the token
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Private-Token", token)

	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ GitLab repository access validated with token!")
		return nil
	}
	return fmt.Errorf("GitLab validation failed: %s", resp.Status)
}

// validateBitbucket validates the repository for Bitbucket, first trying public access.
func validateBitbucket(ctx context.Context, repoURL, token, apiBaseURL string) error {
	// Extract the workspace and repo slug from the URL
	parts := strings.Split(repoURL, "/")
	if len(parts) < 2 { //nolint:gomnd
		return eris.New("invalid bitbucket repository URL")
	}
	workspace := parts[len(parts)-2]
	repoSlug := strings.TrimSuffix(parts[len(parts)-1], ".git")

	// Construct the API request URL
	apiURL := fmt.Sprintf("%s/repositories/%s/%s", apiBaseURL, workspace, repoSlug)

	// First try accessing the repo without a token (public access)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If public access works, return success
	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Bitbucket repository is public and accessible!")
		return nil
	}

	// If public access fails and no token provided, return error
	if token == "" {
		return errors.New("repository is not publicly accessible, please provide an access token")
	}

	// Try again with the token
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Bitbucket repository access validated with token!")
		return nil
	}
	return fmt.Errorf("bitbucket validation failed: %s", resp.Status)
}
