package organization

import (
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

func (h *Handler) saveOrganization(org models.Organization) error {
	h.configService.GetConfig().OrganizationID = org.ID
	err := h.configService.Save()
	if err != nil {
		return eris.Wrap(err, "Failed to save organization")
	}
	return nil
}
