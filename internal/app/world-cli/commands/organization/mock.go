package organization

import (
	"context"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/internal/app/world-cli/interfaces"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

// Interface guard.
var _ interfaces.OrganizationHandler = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Create(ctx context.Context, flags models.CreateOrganizationFlags) (models.Organization, error) {
	args := m.Called(ctx, flags)
	return args.Get(0).(models.Organization), args.Error(1)
}

func (m *MockHandler) Switch(ctx context.Context, flags models.SwitchOrganizationFlags) (models.Organization, error) {
	args := m.Called(ctx, flags)
	return args.Get(0).(models.Organization), args.Error(1)
}

func (m *MockHandler) MembersList(ctx context.Context, org models.Organization, flags models.MembersListFlags) error {
	args := m.Called(ctx, org, flags)
	return args.Error(0)
}

func (m *MockHandler) PromptForSwitch(ctx context.Context, orgs []models.Organization, enableCreation bool,
) (models.Organization, error) {
	args := m.Called(ctx, orgs, enableCreation)
	return args.Get(0).(models.Organization), args.Error(1)
}

func (m *MockHandler) PrintNoOrganizations() {
	m.Called()
}
