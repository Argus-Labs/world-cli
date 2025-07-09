package cardinal

import (
	"context"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/world/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
)

var _ interfaces.CardinalHandler = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Start(ctx context.Context, flags models.StartCardinalFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) Stop(ctx context.Context, flags models.StopCardinalFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) Restart(ctx context.Context, flags models.RestartCardinalFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) Dev(ctx context.Context, flags models.DevCardinalFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) Purge(ctx context.Context, flags models.PurgeCardinalFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) Build(ctx context.Context, flags models.BuildCardinalFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}
