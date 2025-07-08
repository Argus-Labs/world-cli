package user

import (
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/services/input"
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
