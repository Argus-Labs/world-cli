package organization

import (
	"context"

	"pkg.world.dev/world-cli/cmd/pkg/clients/api"
	"pkg.world.dev/world-cli/cmd/pkg/clients/config"
	"pkg.world.dev/world-cli/cmd/pkg/clients/input"
	"pkg.world.dev/world-cli/cmd/pkg/models"
)

// Interface guard.
var _ HandlerInterface = (*Handler)(nil)

type Handler struct {
	projectHandler ProjectHandler
	inputClient    input.ClientInterface
	apiClient      api.ClientInterface
	configClient   config.ClientInterface
}

type HandlerInterface interface {
	Create(
		ctx context.Context,
		state *models.CommandState,
		flags models.CreateOrganizationFlags,
	) (models.Organization, error)
	Switch(
		ctx context.Context,
		state *models.CommandState,
		flags models.SwitchOrganizationFlags,
	) (models.Organization, error)
	PromptForSwitch(
		ctx context.Context,
		state *models.CommandState,
		orgs []models.Organization,
		createNew bool,
	) (models.Organization, error)
}

type ProjectHandler interface {
	Switch(
		ctx context.Context,
		state *models.CommandState,
		flags models.SwitchProjectFlags,
		createNew bool,
	) (models.Project, error)
}

func NewHandler(
	projectHandler ProjectHandler,
	inputClient input.ClientInterface,
	apiClient api.ClientInterface,
	configClient config.ClientInterface,
) HandlerInterface {
	return &Handler{
		projectHandler: projectHandler,
		inputClient:    inputClient,
		apiClient:      apiClient,
		configClient:   configClient,
	}
}
