package cloud

import (
	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
)

// Interface guard.
var _ interfaces.CloudHandler = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Deploy(force bool) error {
	args := m.Called(force)
	return args.Error(0)
}

func (m *MockHandler) Status() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockHandler) Promote() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockHandler) Destroy() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockHandler) Reset() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockHandler) Logs(region string, env string) error {
	args := m.Called(region, env)
	return args.Error(0)
}
