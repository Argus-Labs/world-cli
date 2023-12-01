package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pelletier/go-toml"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	WorldCLIConfigFileEnvVariable = "WORLD_CLI_CONFIG_FILE"
	WorldCLIConfigFilename        = "world.toml"

	flagForConfigFile = "config"
)

var (
	// Items under these toml headers will be included in the environment variables when
	// running docker. An error will be generated if a duplicate key is found across
	// these sections.
	dockerEnvHeaders = []string{"cardinal", "evm"}
)

type Config struct {
	RootDir   string
	Detach    bool
	Build     bool
	Debug     bool
	Timeout   int
	DockerEnv map[string]string
}

func AddConfigFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagForConfigFile, "", "a toml encoded config file")
}

func GetConfig(cmd *cobra.Command) (cfg Config, err error) {
	if !cmd.Flags().Changed(flagForConfigFile) {
		// The config flag was not set. Attempt to find the config via environment variables or in the local directory
		return loadConfig("")
	}
	// The config flag was explicitly set
	configFile, err := cmd.Flags().GetString(flagForConfigFile)
	if err != nil {
		return cfg, err
	}
	if configFile == "" {
		return cfg, errors.New("config cannot be empty")
	}
	return loadConfig(configFile)
}

func loadConfig(filename string) (Config, error) {
	if filename != "" {
		return loadConfigFromFile(filename)
	}
	// Was the file set as an environment variable
	if filename = os.Getenv(WorldCLIConfigFileEnvVariable); filename != "" {
		return loadConfigFromFile(filename)
	}
	// Is there a config in this local directory?
	currDir, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}

	for {
		filename = path.Join(currDir, WorldCLIConfigFilename)
		if cfg, err := loadConfigFromFile(filename); err == nil {
			return cfg, nil
		} else if !os.IsNotExist(err) {
			return cfg, err
		}
		before := currDir
		currDir = path.Join(currDir, "..")
		if currDir == before {
			break
		}
	}

	return Config{}, errors.New("no config file found")
}

func loadConfigFromFile(filename string) (Config, error) {
	cfg := Config{
		DockerEnv: map[string]string{},
	}
	file, err := os.Open(filename)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	data := map[string]any{}
	if err = toml.NewDecoder(file).Decode(&data); err != nil {
		return cfg, err
	}
	if rootDir, ok := data["root_dir"]; ok {
		cfg.RootDir = rootDir.(string)
	} else {
		cfg.RootDir, _ = filepath.Split(filename)
	}

	for _, header := range dockerEnvHeaders {
		m, ok := data[header]
		if !ok {
			continue
		}
		for key, val := range m.(map[string]any) {
			if _, ok := cfg.DockerEnv[key]; ok {
				return cfg, fmt.Errorf("duplicate env variable %q", key)
			}
			cfg.DockerEnv[key] = fmt.Sprintf("%v", val)
		}
	}

	log.Debug().Msgf("successfully loaded config from %q", filename)

	return cfg, nil
}
