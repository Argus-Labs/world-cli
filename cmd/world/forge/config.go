package forge

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
	commonConfig "pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
)

// TODO: break this config into credentials and known projects. Don't save org/project id in the config.
// consider adding a .forge directory with project config alongside the world.toml file.
const (
	forgeConfigFileName = "forgeconfig.json"
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

func GetForgeConfig() (Config, error) {
	var config Config

	fullConfigDir, err := commonConfig.GetCLIConfigDir()
	if err != nil {
		return config, err
	}

	configFile := filepath.Join(fullConfigDir, forgeConfigFileName)

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

func FindGitPathAndURL() (string, string, error) {
	urlData, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return "", "", err
	}
	url := strings.TrimSpace(string(urlData))
	url = replaceLast(url, ".git", "")
	workingDir, err := os.Getwd()
	if err != nil {
		return "", url, err
	}
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", url, err
	}
	rootPath := strings.TrimSpace(string(root))
	path := strings.Replace(workingDir, rootPath, "", 1)
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	return path, url, nil
}

func SaveForgeConfig(globalConfig Config) error {
	fullConfigDir, err := commonConfig.GetCLIConfigDir()
	if err != nil {
		return eris.Wrap(err, "failed to get config dir")
	}

	configFile := filepath.Join(fullConfigDir, forgeConfigFileName)

	configJSON, err := json.Marshal(globalConfig)
	if err != nil {
		return eris.Wrap(err, "failed to marshal config")
	}

	return os.WriteFile(configFile, configJSON, 0600)
}
