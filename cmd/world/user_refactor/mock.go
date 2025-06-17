package user

import (
	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/world/pkg/models"
)

// Interface guard.
var _ HandlerInterface = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) InviteToOrganization(
	ctx models.CommandContext,
	flags *models.InviteUserToOrganizationFlags,
) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) ChangeRoleInOrganization(
	ctx models.CommandContext,
	flags *models.ChangeUserRoleInOrganizationFlags,
) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) Update(ctx models.CommandContext, flags *models.UpdateUserFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}
