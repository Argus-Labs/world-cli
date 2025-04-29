package ports

import "context"

// ForgeProject defines the interface for a world forge project.
type ForgeProject interface {
	// Create creates a new Forge project.
	CreateProject(ctx context.Context) error
}

// ProjectService is the global instance that implements ForgeProject
var ProjectService ForgeProject
