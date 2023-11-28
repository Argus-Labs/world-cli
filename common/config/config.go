package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pelletier/go-toml"
	"github.com/rs/zerolog/log"
)

const (
	WorldCLIConfigFileEnvVariable = "WORLD_CLI_CONFIG_FILE"
	WorldCLIConfigFilename        = "world.toml"
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

func LoadConfig(filename string) (Config, error) {
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
