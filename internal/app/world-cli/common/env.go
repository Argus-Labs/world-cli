package common

import (
	"os"

	"github.com/rotisserie/eris"
)

// WithEnv sets the environment variables from the given map.
func WithEnv(env map[string]string) error {
	for key, value := range env {
		if err := os.Setenv(key, value); err != nil {
			return eris.Wrap(err, "Failed to set env var")
		}
	}
	return nil
}
