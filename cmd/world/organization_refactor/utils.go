package organization

import (
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/pkg/models"
)

func (h *Handler) saveOrganization(org models.Organization) error {
	h.configClient.GetConfig().OrganizationID = org.ID
	err := h.configClient.Save()
	if err != nil {
		return eris.Wrap(err, "Failed to save organization")
	}
	return nil
}
