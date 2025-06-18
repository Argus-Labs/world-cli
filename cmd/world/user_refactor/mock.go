package user

import (
	"context"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

// Interface guard.
var _ interfaces.UserHandler = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) InviteToOrganization(
	ctx context.Context,
	flags *models.InviteUserToOrganizationFlags,
) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) ChangeRoleInOrganization(
	ctx context.Context,
	flags *models.ChangeUserRoleInOrganizationFlags,
) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}

func (m *MockHandler) Update(ctx context.Context, flags *models.UpdateUserFlags) error {
	args := m.Called(ctx, flags)
	return args.Error(0)
}
