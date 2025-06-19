package project

import (
	"context"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/pkg/models"
)

// Interface guard.
var _ HandlerInterface = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Create(
	ctx context.Context,
	state *models.CommandState,
	flags models.CreateProjectFlags,
) (models.Project, error) {
	args := m.Called(ctx, state, flags)
	return args.Get(0).(models.Project), args.Error(1)
}

func (m *MockHandler) Switch(
	ctx context.Context,
	state *models.CommandState,
	flags models.SwitchProjectFlags,
	createNew bool,
) (models.Project, error) {
	args := m.Called(ctx, state, flags, createNew)
	return args.Get(0).(models.Project), args.Error(1)
}

func (m *MockHandler) Update(
	ctx context.Context,
	state *models.CommandState,
	flags models.UpdateProjectFlags,
) error {
	args := m.Called(ctx, state, flags)
	return args.Error(0)
}

func (m *MockHandler) Delete(
	ctx context.Context,
	state *models.CommandState,
) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockHandler) ProjectPreCreateUpdateValidation() (string, string, error) {
	args := m.Called()
	return args.Get(0).(string), args.Get(1).(string), args.Error(2)
}

func (m *MockHandler) PromptForProject(
	ctx context.Context,
	projects []models.Project,
	createNew bool,
) (models.Project, error) {
	args := m.Called(ctx, projects, createNew)
	return args.Get(0).(models.Project), args.Error(1)
}
