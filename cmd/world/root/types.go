package root

import (
	"pkg.world.dev/world-cli/cmd/world/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/world/internal/clients/browser"
	"pkg.world.dev/world-cli/cmd/world/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/world/internal/services/config"
)

// Interface guard.
var _ interfaces.RootHandler = (*Handler)(nil)

type Handler struct {
	AppVersion      string
	configService   config.ServiceInterface
	apiClient       api.ClientInterface
	setupController interfaces.CommandSetupController
	browserClient   browser.ClientInterface
}

func NewHandler(
	appVersion string,
	configService config.ServiceInterface,
	apiClient api.ClientInterface,
	setupController interfaces.CommandSetupController,
	browserClient browser.ClientInterface,
) *Handler {
	return &Handler{
		AppVersion:      appVersion,
		configService:   configService,
		apiClient:       apiClient,
		setupController: setupController,
		browserClient:   browserClient,
	}
}
