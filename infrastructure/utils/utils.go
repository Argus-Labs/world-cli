package utils

import (
	"fmt"
	"net"
	"os"

	"github.com/rotisserie/eris"
)

// FindUnusedPort finds an unused port in the given range
func FindUnusedPort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, eris.New("no available ports in range")
}

// WithEnv sets environment variables from a map
func WithEnv(env map[string]string) error {
	for key, value := range env {
		if err := os.Setenv(key, value); err != nil {
			return eris.Wrapf(err, "failed to set environment variable %s", key)
		}
	}
	return nil
}
