package project

import (
	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/world/pkg/models"
)

// Interface guard.
var _ HandlerInterface = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Create(ctx models.CommandContext, flags *models.CreateProjectFlags, createNew bool,
) (*models.Project, error) {
	args := m.Called(ctx, flags, createNew)
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockHandler) Switch(ctx models.CommandContext, flags *models.SwitchProjectFlags,
) (*models.Project, error) {
	args := m.Called(ctx, flags)
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockHandler) Update(ctx models.CommandContext, flags *models.UpdateProjectFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) Delete(ctx models.CommandContext) error {
	args := m.Called(ctx)
	return args.Error(0)
}
