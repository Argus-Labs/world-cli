package user

import (
	"context"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/world/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
)

// Interface guard.
var _ interfaces.UserHandler = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) InviteToOrganization(
	ctx context.Context,
	organization models.Organization,
	flags models.InviteUserToOrganizationFlags,
) error {
	args := m.Called(ctx, organization, flags)
	return args.Error(0)
}

func (m *MockHandler) ChangeRoleInOrganization(
	ctx context.Context,
	organization models.Organization,
	flags models.ChangeUserRoleInOrganizationFlags,
) error {
	args := m.Called(ctx, organization, flags)
	return args.Error(0)
}

func (m *MockHandler) Update(ctx context.Context, flags models.UpdateUserFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}
