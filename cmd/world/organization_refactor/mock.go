package organization

import (
	"context"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

// Interface guard.
var _ interfaces.OrganizationHandler = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Create(ctx context.Context, flags *models.CreateOrganizationFlags,
) (models.Organization, error) {
	args := m.Called(ctx, flags)
	return args.Get(0).(models.Organization), args.Error(1)
}

func (m *MockHandler) Switch(ctx context.Context, flags *models.SwitchOrganizationFlags,
) (models.Organization, error) {
	args := m.Called(ctx, flags)
	return args.Get(0).(models.Organization), args.Error(1)
}
