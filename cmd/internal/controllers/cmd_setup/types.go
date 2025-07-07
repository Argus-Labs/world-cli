package cmdsetup

import (
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/clients/repo"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/cmd/internal/services/input"
)

var (
	ErrLogin = eris.New("not logged in")
)

// Dependencies holds all initialized clients and handlers.
type Dependencies struct {
	ConfigService       config.ServiceInterface
	InputService        input.ServiceInterface
	APIClient           api.ClientInterface
	RepoClient          repo.ClientInterface
	OrganizationHandler interfaces.OrganizationHandler
	ProjectHandler      interfaces.ProjectHandler
	UserHandler         interfaces.UserHandler
	RootHandler         interfaces.RootHandler
	CloudHandler        interfaces.CloudHandler
	EVMHandler          interfaces.EVMHandler
	SetupController     interfaces.CommandSetupController
}

type Controller struct {
	configService       config.ServiceInterface
	inputService        input.ServiceInterface
	apiClient           api.ClientInterface
	repoClient          repo.ClientInterface
	organizationHandler interfaces.OrganizationHandler
	projectHandler      interfaces.ProjectHandler
}

func NewController(
	configService config.ServiceInterface,
	repoClient repo.ClientInterface,
	organizationHandler interfaces.OrganizationHandler,
	projectHandler interfaces.ProjectHandler,
	apiClient api.ClientInterface,
	inputService input.ServiceInterface,
) interfaces.CommandSetupController {
	return &Controller{
		configService:       configService,
		inputService:        inputService,
		repoClient:          repoClient,
		organizationHandler: organizationHandler,
		projectHandler:      projectHandler,
		apiClient:           apiClient,
	}
}
