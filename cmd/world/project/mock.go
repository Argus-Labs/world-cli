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

func (m *MockHandler) Create(
	ctx context.Context,
	flags models.CreateProjectFlags,
) (models.Project, error) {
	args := m.Called(ctx, flags)
	return args.Get(0).(models.Project), args.Error(1)
}

func (m *MockHandler) Switch(
	ctx context.Context,
	flags models.SwitchProjectFlags,
	createNew bool,
) (models.Project, error) {
	args := m.Called(ctx, flags, createNew)
	return args.Get(0).(models.Project), args.Error(1)
}

func (m *MockHandler) Update(
	ctx context.Context,
	project models.Project,
	flags models.UpdateProjectFlags,
) error {
	args := m.Called(ctx, project, flags)
	return args.Error(0)
}

func (m *MockHandler) Delete(
	ctx context.Context,
	project models.Project,
) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockHandler) PreCreateUpdateValidation() (string, string, error) {
	args := m.Called()
	return args.Get(0).(string), args.Get(1).(string), args.Error(2)
}
