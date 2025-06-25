package project

import (
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/clients/repo"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/cmd/internal/services/input"
)

var nilUUID = "00000000-0000-0000-0000-000000000000"

// Interface guard.
var _ interfaces.ProjectHandler = (*Handler)(nil)

type Handler struct {
	repoClient    repo.ClientInterface
	configService config.ServiceInterface
	apiClient     api.ClientInterface
	inputService  input.ServiceInterface
}

// notificationConfig holds common notification configuration fields.
type notificationConfig struct {
	name      string // "Discord" or "Slack"
	tokenName string // What to call the token ("bot token" or "token")
}

func NewHandler(
	repoClient repo.ClientInterface,
	configService config.ServiceInterface,
	apiClient api.ClientInterface,
	inputService input.ServiceInterface,
) interfaces.ProjectHandler {
	return &Handler{
		repoClient:    repoClient,
		configService: configService,
		apiClient:     apiClient,
		inputService:  inputService,
	}
}
