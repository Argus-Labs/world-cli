package project

import (
	"context"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

// Interface guard.
var _ interfaces.ProjectHandler = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Create(ctx context.Context, flags *models.CreateProjectFlags, createNew bool,
) (models.Project, error) {
	args := m.Called(ctx, flags, createNew)
	return args.Get(0).(models.Project), args.Error(1)
}

func (m *MockHandler) Switch(ctx context.Context, flags *models.SwitchProjectFlags,
) (models.Project, error) {
	args := m.Called(ctx, flags)
	return args.Get(0).(models.Project), args.Error(1)
}

func (m *MockHandler) Update(ctx context.Context, flags *models.UpdateProjectFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) Delete(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
