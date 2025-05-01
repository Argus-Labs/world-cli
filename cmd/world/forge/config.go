package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

type ForgeConfig struct {
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

func GetForgeConfig() (ForgeConfig, error) {
	var config ForgeConfig

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

func GetCurrentForgeConfig() (ForgeConfig, error) {
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

func GetCurrentForgeConfigWithContext(ctx context.Context) (*ForgeConfig, error) {
	currConfig, err := GetCurrentForgeConfig()
	// we don't care if we got an error, we will just return it later
	if !currConfig.CurrRepoKnown && //nolint: nestif // not too complex
		currConfig.Credential.Token != "" &&
		currConfig.CurrRepoURL != "" {
		// needed a lookup, and have a token (so we should be logged in)
		// get the organization and project from the project's URL and path
		deployURL := fmt.Sprintf("%s/api/project/?url=%s&path=%s",
			baseURL, url.QueryEscape(currConfig.CurrRepoURL), url.QueryEscape(currConfig.CurrRepoPath))
		body, err := sendRequest(ctx, http.MethodGet, deployURL, nil)
		if err != nil {
			fmt.Println("⚠️ Warning: Failed to lookup World Forge project for Git Repo",
				currConfig.CurrRepoURL, "and path", currConfig.CurrRepoPath, ":", err)
			return &currConfig, err
		}

		// Parse response
		proj, err := parseResponse[project](body)
		if err != nil && err.Error() != "Missing data field in response" {
			// missing data field in response just means nothing was found
			fmt.Println("⚠️ Warning: Failed to parse project lookup response: ", err)
			return &currConfig, err
		}
		if proj != nil {
			// add to list of known projects
			currConfig.KnownProjects = append(currConfig.KnownProjects, KnownProject{
				ProjectID:      proj.ID,
				OrganizationID: proj.OrgID,
				RepoURL:        proj.RepoURL,
				RepoPath:       proj.RepoPath,
				ProjectName:    proj.Name,
			})
			// save the config, but don't change the default ProjectID & OrgID
			err := SaveForgeConfig(currConfig)
			if err != nil {
				fmt.Println("⚠️ Warning: Failed to save config: ", err)
				// continue on, this is not fatal
			}
			// now return a copy of it with the looked up ProjectID and OrganizationID set
			currConfig.ProjectID = proj.ID
			currConfig.OrganizationID = proj.OrgID
			currConfig.CurrProjectName = proj.Name
			currConfig.CurrRepoKnown = true
		}
	}
	return &currConfig, err
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

func SaveForgeConfig(globalConfig ForgeConfig) error {
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

// this is a variable so we can change it for testing login.
var getCurrentForgeConfigWithContext = GetCurrentForgeConfigWithContext
