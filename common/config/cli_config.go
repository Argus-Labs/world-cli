package config

import (
	"os"
	"path/filepath"
)

const (
	configDir = ".worldcli"
)

//nolint:gochecknoglobals // Ok as global, Only used in test during setup and will not interfere with parallel tests.
var GetCLIConfigDir = func() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, configDir), nil
}

func SetupCLIConfigDir() error {
	fullConfigDir, err := GetCLIConfigDir()
	if err != nil {
		return err
	}

	fs, err := os.Stat(fullConfigDir)
	if !os.IsNotExist(err) {
		// remove old .worldcli file
		if !fs.IsDir() {
			err = os.Remove(fullConfigDir)
			if err != nil {
				return err
			}
		}
	}

	return os.MkdirAll(fullConfigDir, 0755)
}
