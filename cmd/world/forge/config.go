package forge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
	commonConfig "pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
)

// TODO: break this config into credentials and known projects. Don't save org/project id in the config.
// consider adding a .forge directory with project config alongside the world.toml file.
const (
	EnvLocal = "LOCAL"
	EnvDev   = "DEV"
	EnvProd  = "PROD"
)

type Config struct {
	OrganizationID string         `json:"organization_id"`
	ProjectID      string         `json:"project_id"`
	Credential     Credential     `json:"credential"`
	KnownProjects  []KnownProject `json:"known_projects"`
	// the following are not saved in json
	// TODO: get rid of these since they will be handled by the init flow state
	CurrRepoKnown   bool   `json:"-"` // when true, the current repo and path are already in known_projects
	CurrRepoURL     string `json:"-"`
	CurrRepoPath    string `json:"-"`
	CurrProjectName string `json:"-"`
}

func getConfigFileName() (string, error) {
	fileName := "forge-config.json"
	if Env == EnvDev || Env == EnvLocal {
		fileName = strings.ToLower(Env) + "-" + fileName
	}
	fullConfigDir, err := commonConfig.GetCLIConfigDir()
	if err != nil {
		return "", eris.Wrap(err, "failed get config dir")
	}
	configFile := filepath.Join(fullConfigDir, fileName)
	return configFile, nil
}

func GetForgeConfig() (Config, error) {
	var config Config

	configFile, err := getConfigFileName()
	if err != nil {
		return config, eris.Wrap(err, "failed get config file name")
	}

	file, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // this is ok, just create empty config
		}
		return config, eris.Wrap(err, "failed to read config file")
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
	config.CurrProjectName = ""
	return config, nil
}

func GetCurrentForgeConfig() (Config, error) {
	currConfig, err := GetForgeConfig()
	// we deliberately ignore any error here and just return it at the end
	// so that we can fill out and much info as we do have
	currConfig.CurrRepoKnown = false
	currConfig.CurrRepoPath, currConfig.CurrRepoURL, _ = FindGitPathAndURL()
	if currConfig.CurrRepoURL != "" {
		for i := range currConfig.KnownProjects {
			knownProject := currConfig.KnownProjects[i]
			if knownProject.RepoURL == currConfig.CurrRepoURL && knownProject.RepoPath == currConfig.CurrRepoPath {
				currConfig.ProjectID = knownProject.ProjectID
				currConfig.OrganizationID = knownProject.OrganizationID
				currConfig.CurrProjectName = knownProject.ProjectName
				currConfig.CurrRepoKnown = true
				break
			}
		}
	}
	return currConfig, err
}

func SaveForgeConfig(globalConfig Config) error {
	configFile, err := getConfigFileName()
	if err != nil {
		return eris.Wrap(err, "failed get config file name")
	}

	configJSON, err := json.Marshal(globalConfig)
	if err != nil {
		return eris.Wrap(err, "failed to marshal config")
	}

	return os.WriteFile(configFile, configJSON, 0600)
}
