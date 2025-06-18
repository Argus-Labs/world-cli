package repo

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ ClientInterface = (*MockClient)(nil)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) FindGitPathAndURL() (string, string, error) {
	args := m.Called()
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockClient) ValidateRepoToken(ctx context.Context, repoURL, token string) error {
	args := m.Called(ctx, repoURL, token)
	return args.Error(0)
}

func (m *MockClient) ValidateRepoPath(ctx context.Context, repoURL, token, path string) error {
	args := m.Called(ctx, repoURL, token, path)
	return args.Error(0)
}
