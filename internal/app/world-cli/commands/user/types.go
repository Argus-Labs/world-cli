package user

import (
	"pkg.world.dev/world-cli/internal/app/world-cli/clients/api"
	"pkg.world.dev/world-cli/internal/app/world-cli/interfaces"
	"pkg.world.dev/world-cli/internal/app/world-cli/services/input"
)

// Interface guard.
var _ interfaces.UserHandler = (*Handler)(nil)

type Handler struct {
	apiClient    api.ClientInterface
	inputService input.ServiceInterface
}

func NewHandler(apiClient api.ClientInterface, inputService input.ServiceInterface) interfaces.UserHandler {
	return &Handler{
		apiClient:    apiClient,
		inputService: inputService,
	}
}
