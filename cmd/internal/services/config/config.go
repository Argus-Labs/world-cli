package config

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

	defaultFileName = "forge-config.json"
)

var ErrCannotSaveConfig = eris.New("Critical config update error could not save")

func NewService(env string) (ServiceInterface, error) {
	service := &Service{
		Env:    env,
		Config: Config{},
	}

	err := service.getSetConfig()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get config")
	}
	return service, nil
}

func (s *Service) GetConfig() *Config {
	return &s.Config
}

func (s *Service) Save() error {
	configFile, err := s.getConfigFileName()
	if err != nil {
		return eris.Wrap(err, "failed get config file name")
	}

	configJSON, err := json.Marshal(s.Config)
	if err != nil {
		return eris.Wrap(err, "failed to marshal config")
	}

	return os.WriteFile(configFile, configJSON, 0600)
}

func (s *Service) AddKnownProject(
	projectID string,
	projectName string,
	organizationID string,
	repoURL string,
	repoPath string,
) {
	s.Config.KnownProjects = append(s.Config.KnownProjects, KnownProject{
		ProjectID:      projectID,
		ProjectName:    projectName,
		OrganizationID: organizationID,
		RepoURL:        repoURL,
		RepoPath:       repoPath,
	})
}

//////////////////////////////////////////////////////////////////////////////////////////////////
// internal functions
//////////////////////////////////////////////////////////////////////////////////////////////////

func (s *Service) getSetConfig() error {
	var config Config

	configFile, err := s.getConfigFileName()
	if err != nil {
		return eris.Wrap(err, "failed get config file name")
	}

	file, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // this is ok, just create empty config
		}
		return eris.Wrap(err, "failed to read config file")
	}

	// Unmarshal the config
	err = json.Unmarshal(file, &config)
	if err != nil {
		logger.Error(eris.Wrap(err, "failed to unmarshal config"))
		return err
	}
	// these will get set in forge/common.go's GetCurrentConfig()
	config.CurrRepoKnown = false
	config.CurrRepoURL = ""
	config.CurrRepoPath = ""
	config.CurrProjectName = ""

	s.Config = config
	return nil
}

func (s *Service) getConfigFileName() (string, error) {
	fileName := defaultFileName
	if s.Env == EnvDev || s.Env == EnvLocal {
		fileName = strings.ToLower(s.Env) + "-" + fileName
	}
	fullConfigDir, err := commonConfig.GetCLIConfigDir()
	if err != nil {
		return "", eris.Wrap(err, "failed get config dir")
	}
	configFile := filepath.Join(fullConfigDir, fileName)
	return configFile, nil
}
