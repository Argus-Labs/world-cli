package organization

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

func (m *MockHandler) Create(ctx context.Context, state *models.CommandState, flags models.CreateOrganizationFlags,
) (models.Organization, error) {
	args := m.Called(ctx, state, flags)
	return args.Get(0).(models.Organization), args.Error(1)
}

func (m *MockHandler) Switch(ctx context.Context, state *models.CommandState, flags models.SwitchOrganizationFlags,
) (models.Organization, error) {
	args := m.Called(ctx, state, flags)
	return args.Get(0).(models.Organization), args.Error(1)
}

func (m *MockHandler) PromptForSwitch(
	ctx context.Context,
	state *models.CommandState,
	orgs []models.Organization,
	createNew bool,
) (models.Organization, error) {
	args := m.Called(ctx, state, orgs, createNew)
	return args.Get(0).(models.Organization), args.Error(1)
}
