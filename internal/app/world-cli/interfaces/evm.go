package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

type EVMHandler interface {
	Start(ctx context.Context, flags models.StartEVMFlags) error
	Stop(ctx context.Context, flags models.StopEVMFlags) error
}
