package project

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/clients/repo"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/cmd/internal/services/input"
)

var nilUUID = "00000000-0000-0000-0000-000000000000"

// Interface guard.
var _ interfaces.ProjectHandler = (*Handler)(nil)

// RegionSelector interface for selecting regions.
type RegionSelector interface {
	SelectRegions(ctx context.Context, regions []string, selectedRegions []string) ([]string, error)
}

type Handler struct {
	repoClient     repo.ClientInterface
	configService  config.ServiceInterface
	apiClient      api.ClientInterface
	inputService   input.ServiceInterface
	regionSelector RegionSelector
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
		repoClient:     repoClient,
		configService:  configService,
		apiClient:      apiClient,
		inputService:   inputService,
		regionSelector: &BubbleteeRegionSelector{},
	}
}

// NewHandlerWithRegionSelector creates a new project handler with a custom region selector.
// This is used for testing purposes to inject a mock region selector.
func NewHandlerWithRegionSelector(
	repoClient repo.ClientInterface,
	configService config.ServiceInterface,
	apiClient api.ClientInterface,
	inputService input.ServiceInterface,
	regionSelector RegionSelector,
) interfaces.ProjectHandler {
	return &Handler{
		repoClient:     repoClient,
		configService:  configService,
		apiClient:      apiClient,
		inputService:   inputService,
		regionSelector: regionSelector,
	}
}
