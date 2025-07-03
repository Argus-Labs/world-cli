package organization

import (
	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/common/printer"
)

func (h *Handler) saveOrganization(org models.Organization) error {
	h.configService.GetConfig().OrganizationID = org.ID
	err := h.configService.Save()
	if err != nil {
		return eris.Wrap(err, "Failed to save organization: "+org.Name)
	}
	return nil
}

func (h *Handler) PrintNoOrganizations() {
	printer.NewLine(1)
	printer.Headerln("   No Organizations Found   ")
	printer.Info("1. Use ")
	printer.Notification("'world organization create'")
	printer.Infoln(" to create an organization.")
	printer.Info("2. Have a member send invite using ")
	printer.Notification("'world organization invite'")
	printer.Infoln(".")
}
