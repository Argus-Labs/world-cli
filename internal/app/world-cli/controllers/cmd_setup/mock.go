package cmdsetup

import (
	"context"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/internal/app/world-cli/interfaces"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

// Ensure MockController implements the interface.
var _ interfaces.CommandSetupController = (*MockController)(nil)

// MockController is a mock implementation of CommandSetupController.
type MockController struct {
	mock.Mock
}

// SetupCommandState mocks the setup command.
func (m *MockController) SetupCommandState(
	ctx context.Context,
	req models.SetupRequest,
) (models.CommandState, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return models.CommandState{}, args.Error(1)
	}
	return args.Get(0).(models.CommandState), args.Error(1)
}
