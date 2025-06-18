package repo

import "context"

var _ ClientInterface = (*Client)(nil)

type Client struct {
}

type ClientInterface interface {
	FindGitPathAndURL() (string, string, error)
	ValidateRepoToken(ctx context.Context, repoURL, token string) error
	ValidateRepoPath(ctx context.Context, repoURL, token, path string) error
}
