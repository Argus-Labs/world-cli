package forge

import (
	"context"

	"github.com/rotisserie/eris"
	forgeinterface "pkg.world.dev/world-cli/common/forge"
)

// Service implements Forge interface from the common package.
// Allows for external forge operations to be performed without exposing the internal implementation details.
// NOTE: Can extend ports.Forge to create other operations when needed.
type Service struct{}

// Interface Guard
var _ forgeinterface.Forge = &Service{}

// Login will perform the login process for the user, including organization & project creation/selection.
func (s *Service) Login(ctx context.Context) error {
	if err := login(ctx); err != nil {
		return eris.Wrap(err, "Forge service login failed")
	}
	return nil
}
