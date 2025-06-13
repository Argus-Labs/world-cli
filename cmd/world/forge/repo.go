package forge

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/rotisserie/eris"
)

const minimumURLParts = 2

var (
	ErrNotInGitRepository     = eris.New("Not in a git repository")
	ErrNotInWorldCardinalRoot = eris.New("Not in a World Cadinal root")
)

// identifyProvider determines the Git provider based on the URL's host.
func identifyProvider(repoURL string) (string, string, error) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	host := parsedURL.Host
	switch {
	case strings.Contains(host, "github.com"):
		return "GitHub", "https://api.github.com", nil
	case strings.Contains(host, "gitlab.com"):
		return "GitLab", "https://gitlab.com/api/v4", nil
	case strings.Contains(host, "bitbucket.org"):
		return "Bitbucket", "https://api.bitbucket.org/2.0", nil
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

// validateGitHub validates the token and repository for GitHub.
func validateGitHub(ctx context.Context, repoURL, token, apiBaseURL string) error {
	// Extract the owner and repo name from the URL
	parts := strings.Split(repoURL, "/")
	if len(parts) < minimumURLParts {
		return eris.New("invalid github repository URL")
	}
	repo := strings.TrimSuffix(parts[len(parts)-1], ".git")
	owner := parts[len(parts)-2]

	// Construct the API request URL
	apiURL := fmt.Sprintf("%s/repos/%s/%s", apiBaseURL, owner, repo)

	// Make the API request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}

	// Only set authorization header if token is provided
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("GitHub validation failed: %s", resp.Status)
}

// validateGitLab validates the token and repository for GitLab.
func validateGitLab(ctx context.Context, repoURL, token, apiBaseURL string) error {
	// Extract the project path from the URL
	parts := strings.Split(repoURL, "/")
	if len(parts) < minimumURLParts {
		return eris.New("invalid gitlab repository URL")
	}
	projectPath := fmt.Sprintf("%s/%s", parts[len(parts)-2], strings.TrimSuffix(parts[len(parts)-1], ".git"))

	// Construct the API request URL
	apiURL := fmt.Sprintf("%s/projects/%s", apiBaseURL, url.QueryEscape(projectPath))

	// Make the API request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}

	// Only set token header if token is provided
	if token != "" {
		req.Header.Set("Private-Token", token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("GitLab validation failed: %s", resp.Status)
}

// validateBitbucket validates the token and repository for Bitbucket.
func validateBitbucket(ctx context.Context, repoURL, token, apiBaseURL string) error {
	// Extract the workspace and repo slug from the URL
	parts := strings.Split(repoURL, "/")
	if len(parts) < minimumURLParts {
		return eris.New("invalid bitbucket repository URL")
	}
	workspace := parts[len(parts)-2]
	repoSlug := strings.TrimSuffix(parts[len(parts)-1], ".git")

	// Construct the API request URL
	apiURL := fmt.Sprintf("%s/repositories/%s/%s", apiBaseURL, workspace, repoSlug)

	// Make the API request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}

	// Only set authorization header if token is provided
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("bitbucket validation failed: %s", resp.Status)
}

func FindGitPathAndURL() (string, string, error) {
	// Try to get the 'origin' remote URL first
	urlData, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil || strings.TrimSpace(string(urlData)) == "" {
		// Fallback: get the first available remote URL
		remoteList, fallbackErr := exec.Command("git", "remote", "-v").Output()
		if fallbackErr != nil {
			return "", "", eris.Wrap(
				fallbackErr,
				"Need to be in git repo: failed to get remote list",
			)
		}
		lines := strings.Split(string(remoteList), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				urlData = []byte(parts[1])
				break
			}
		}
	}

	url := strings.TrimSpace(string(urlData))
	if url == "" {
		return "", "", ErrNotInGitRepository
	}
	url = replaceLast(url, ".git", "")
	workingDir, err := os.Getwd()
	if err != nil {
		return "", url, err
	}
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", url, err
	}
	rootPath := strings.TrimSpace(string(root))
	path := strings.Replace(workingDir, rootPath, "", 1)
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	return path, url, nil
}
