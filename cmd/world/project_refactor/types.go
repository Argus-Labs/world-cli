package project

import (
	"context"

	"pkg.world.dev/world-cli/cmd/pkg/clients/api"
	"pkg.world.dev/world-cli/cmd/pkg/clients/config"
	"pkg.world.dev/world-cli/cmd/pkg/clients/input"
	"pkg.world.dev/world-cli/cmd/pkg/clients/repo"
	"pkg.world.dev/world-cli/cmd/pkg/models"
)

var nilUUID = "00000000-0000-0000-0000-000000000000"

// Interface guard.
var _ HandlerInterface = (*Handler)(nil)

type Handler struct {
	repoClient   repo.ClientInterface
	configClient config.ClientInterface
	apiClient    api.ClientInterface
	inputClient  input.ClientInterface
}

type HandlerInterface interface {
	Create(ctx context.Context, state *models.CommandState, flags models.CreateProjectFlags) (models.Project, error)
	Switch(
		ctx context.Context,
		state *models.CommandState,
		flags models.SwitchProjectFlags,
		createNew bool,
	) (models.Project, error)
	Update(ctx context.Context, state *models.CommandState, flags models.UpdateProjectFlags) error
	Delete(ctx context.Context, state *models.CommandState) error

	ProjectPreCreateUpdateValidation() (string, string, error)
}

// notificationConfig holds common notification configuration fields.
type notificationConfig struct {
	name      string // "Discord" or "Slack"
	tokenName string // What to call the token ("bot token" or "token")
}

func NewHandler(
	repoClient repo.ClientInterface,
	configClient config.ClientInterface,
	apiClient api.ClientInterface,
	inputClient input.ClientInterface,
) HandlerInterface {
	return &Handler{
		repoClient:   repoClient,
		configClient: configClient,
		apiClient:    apiClient,
		inputClient:  inputClient,
	}
}
