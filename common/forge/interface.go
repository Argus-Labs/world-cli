package forge

import "context"

// Forge defines the interface for common forge operations.
type Forge interface {
	// Login will perform the login process for the user, including organization & project creation/selection.
	Login(ctx context.Context) error
}

// Service is the global instance that implements Forge.
var Service Forge
