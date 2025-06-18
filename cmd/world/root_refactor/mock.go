package root

import (
	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/world/pkg/models"
)

// Interface guard.
var _ HandlerInterface = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Create(directory string) error {
	args := m.Called(directory)
	return args.Error(0)
}

func (m *MockHandler) Doctor() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockHandler) Version(check bool) error {
	args := m.Called(check)
	return args.Error(0)
}

func (m *MockHandler) Login(ctx models.CommandContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}
