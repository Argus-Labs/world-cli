package evm

import (
	"context"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/internal/app/world-cli/interfaces"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

var _ interfaces.EVMHandler = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Start(ctx context.Context, flags models.StartEVMFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) Stop(ctx context.Context, flags models.StopEVMFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}
