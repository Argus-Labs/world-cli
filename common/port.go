package common

import (
	"fmt"
	"net"
)

// FindUnusedPort finds an unused port in the range [start, end] for Cardinal Editor
func FindUnusedPort(start int, end int) (int, error) {
	for port := start; port <= end; port++ {
		address := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			if err := listener.Close(); err != nil {
				return 0, err
			}
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port in the range %d-%d", start, end)
}
