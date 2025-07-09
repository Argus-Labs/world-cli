package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/cmd/world/internal/models"
)

type EVMHandler interface {
	Start(ctx context.Context, flags models.StartEVMFlags) error
	Stop(ctx context.Context, flags models.StopEVMFlags) error
}
