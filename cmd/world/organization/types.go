package organization

import (
	"pkg.world.dev/world-cli/cmd/world/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/world/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/world/internal/services/config"
	"pkg.world.dev/world-cli/cmd/world/internal/services/input"
)

// Interface guard.
var _ interfaces.OrganizationHandler = (*Handler)(nil)

type Handler struct {
	projectHandler interfaces.ProjectHandler
	inputService   input.ServiceInterface
	apiClient      api.ClientInterface
	configService  config.ServiceInterface
}

func NewHandler(
	projectHandler interfaces.ProjectHandler,
	inputService input.ServiceInterface,
	apiClient api.ClientInterface,
	configService config.ServiceInterface,
) interfaces.OrganizationHandler {
	return &Handler{
		projectHandler: projectHandler,
		inputService:   inputService,
		apiClient:      apiClient,
		configService:  configService,
	}
}
