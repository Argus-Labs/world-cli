package globalconfig

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/logger"
)

const (
	configDir          = ".worldcli"
	credentialFileName = "credential.json" //nolint:gosec // This is not a credential
)

type Credential struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, configDir), nil
}

func GetWorldForgeCredential() (Credential, error) {
	var cred Credential

	fullConfigDir, err := GetConfigDir()
	if err != nil {
		return cred, err
	}

	tokenFile := filepath.Join(fullConfigDir, credentialFileName)

	file, err := os.ReadFile(tokenFile)
	if err != nil {
		return cred, err
	}

	// Unmarshal the token
	err = json.Unmarshal(file, &cred)
	if err != nil {
		logger.Error(eris.Wrap(err, "failed to unmarshal token"))
		return cred, err
	}

	return cred, nil
}

func SetWorldForgeToken(name string, token string) error {
	fullConfigDir, err := GetConfigDir()
	if err != nil {
		return eris.Wrap(err, "failed to get config dir")
	}

	tokenFile := filepath.Join(fullConfigDir, credentialFileName)

	cred := Credential{
		Token: token,
		Name:  name,
	}

	credJSON, err := json.Marshal(cred)
	if err != nil {
		return eris.Wrap(err, "failed to marshal token")
	}

	return os.WriteFile(tokenFile, credJSON, 0600)
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
