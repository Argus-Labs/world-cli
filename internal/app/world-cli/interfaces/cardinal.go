package interfaces

import (
	"context"

	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

type CardinalHandler interface {
	Start(ctx context.Context, f models.StartCardinalFlags) error
	Stop(ctx context.Context, f models.StopCardinalFlags) error
	Restart(ctx context.Context, f models.RestartCardinalFlags) error
	Dev(ctx context.Context, f models.DevCardinalFlags) error
	Purge(ctx context.Context, f models.PurgeCardinalFlags) error
	Build(ctx context.Context, f models.BuildCardinalFlags) error
}
