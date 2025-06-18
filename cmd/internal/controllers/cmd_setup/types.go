package cmdsetup

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/clients/repo"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
)

var (
	ErrLogin = eris.New("not logged in")
)

type Controller struct {
	configService       config.ServiceInterface
	repoClient          repo.ClientInterface
	organizationHandler interfaces.OrganizationHandler
	projectHandler      interfaces.ProjectHandler
	apiClient           APIClientInterface // TODO: Implement API package
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
