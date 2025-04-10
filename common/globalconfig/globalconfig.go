package globalconfig

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/logger"
)

const (
	configDir            = ".worldcli"
	globalConfigFileName = "config.json"
)

var (
	// Env is the environment the CLI is running in
	Env = "DEV"
)

var GetConfigDir = func() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, configDir), nil
}

type Credential struct {
	Token string `json:"token"`
	ID    string `json:"id"`
	Name  string `json:"name"`
}

type KnownProject struct {
	RepoURL        string `json:"repo_url"`
	RepoPath       string `json:"repo_path"`
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
}

type GlobalConfig struct {
	OrganizationID string         `json:"organization_id"`
	ProjectID      string         `json:"project_id"`
	Credential     Credential     `json:"credential"`
	KnownProjects  []KnownProject `json:"known_projects"`
	// the following are not saved in json
	CurrRepoKnown bool   `json:""` // when true, the current repo and path are already in known_projects
	CurrRepoURL   string `json:""`
	CurrRepoPath  string `json:""`
}

func GetGlobalConfig() (GlobalConfig, error) {
	var config GlobalConfig

	fullConfigDir, err := GetConfigDir()
	if err != nil {
		return config, err
	}

	configFile := filepath.Join(fullConfigDir, globalConfigFileName)

	file, err := os.ReadFile(configFile)
	if err != nil {
		return config, err
	}

	// Unmarshal the config
	err = json.Unmarshal(file, &config)
	if err != nil {
		logger.Error(eris.Wrap(err, "failed to unmarshal config"))
		return config, err
	}
	// these will get set in forge/common.go's GetCurrentConfig()
	config.CurrRepoKnown = false
	config.CurrRepoURL = ""
	config.CurrRepoPath = ""
	return config, nil
}

func SaveGlobalConfig(globalConfig GlobalConfig) error {
	fullConfigDir, err := GetConfigDir()
	if err != nil {
		return eris.Wrap(err, "failed to get config dir")
	}

	configFile := filepath.Join(fullConfigDir, globalConfigFileName)

	configJSON, err := json.Marshal(globalConfig)
	if err != nil {
		return eris.Wrap(err, "failed to marshal config")
	}

	return os.WriteFile(configFile, configJSON, 0600)
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
