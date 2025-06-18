package organization

import (
	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/world/pkg/models"
)

// Interface guard.
var _ HandlerInterface = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Create(ctx models.CommandContext, flags *models.CreateOrganizationFlags,
) (*models.Organization, error) {
	args := m.Called(ctx, flags)
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockHandler) Switch(ctx models.CommandContext, flags *models.SwitchOrganizationFlags,
) (*models.Organization, error) {
	args := m.Called(ctx, flags)
	return args.Get(0).(*models.Organization), args.Error(1)
}
