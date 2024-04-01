package globalconfig

import (
	"os"
	"path/filepath"
)

const (
	configDir = ".worldcli"
)

func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, configDir), nil
}

func SetupConfigDir() error {
	fullConfigDir, err := GetConfigDir()
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
