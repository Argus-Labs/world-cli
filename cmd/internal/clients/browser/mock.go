package browser

import "github.com/stretchr/testify/mock"

var _ ClientInterface = (*MockClient)(nil)

// MockClient is a mock implementation of ClientInterface for testing.
type MockClient struct {
	mock.Mock
}

// OpenURL mocks opening a URL in the browser.
func (m *MockClient) OpenURL(url string) error {
	args := m.Called(url)
	return args.Error(0)
}
