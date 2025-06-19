package cmdsetup

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/pkg/clients/config"
	"pkg.world.dev/world-cli/cmd/pkg/clients/input"
	"pkg.world.dev/world-cli/cmd/pkg/clients/repo"
	"pkg.world.dev/world-cli/cmd/pkg/models"
)

var (
	ErrLogin = eris.New("not logged in")
)

// Service implements the setup service interface.
type Service struct {
	configClient        config.ClientInterface
	repoClient          repo.ClientInterface
	organizationHandler OrganizationHandler
	projectHandler      ProjectHandler
	apiClient           APIClientInterface
	inputClient         input.ClientInterface
}

// APIClientInterface defines the API operations needed by the setup service.
type APIClientInterface interface {
	GetUser(ctx context.Context) (models.User, error)
	GetOrganizations(ctx context.Context) ([]models.Organization, error)
	GetOrganizationsInvitedTo(ctx context.Context) ([]models.Organization, error)
	AcceptOrganizationInvitation(ctx context.Context, orgID string) error
	GetProjects(ctx context.Context, orgID string) ([]models.Project, error)
	LookupProjectFromRepo(ctx context.Context, repoURL, repoPath string) (models.Project, error)
	GetOrganizationByID(ctx context.Context, id string) (models.Organization, error)
	GetProjectByID(ctx context.Context, id string) (models.Project, error)
}

type OrganizationHandler interface {
	PromptForSwitch(
		ctx context.Context,
		state *models.CommandState,
		orgs []models.Organization,
		createNew bool,
	) (models.Organization, error)
	Create(
		ctx context.Context,
		state *models.CommandState,
		flags models.CreateOrganizationFlags,
	) (models.Organization, error)
}

type ProjectHandler interface {
	Switch(
		ctx context.Context,
		state *models.CommandState,
		flags models.SwitchProjectFlags,
		createNew bool,
	) (models.Project, error)
	Create(ctx context.Context, state *models.CommandState, flags models.CreateProjectFlags) (models.Project, error)
	ProjectPreCreateUpdateValidation() (string, string, error)
}
