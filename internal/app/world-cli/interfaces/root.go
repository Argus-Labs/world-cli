package interfaces

import "context"

// RootHandler defines the interface for root-level CLI operations.
type RootHandler interface {
	// Create initializes a new World project in the specified directory.
	Create(directory string) error

	// Doctor performs system diagnostics and validation checks.
	Doctor() error

	// Version displays version information and optionally checks for updates.
	Version(check bool) error

	// Login handles user authentication and login flow.
	Login(ctx context.Context) error

	// SetAppVersion sets the application version for display purposes.
	SetAppVersion(version string)
}
