package config

import (
	"errors"
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

type Config struct {
	RootDir  string
	Build    bool `toml:"build"`
	Debug    bool `toml:"debug"`
	Detach   bool `toml:"detach"`
	Timeout  int  `toml:"timeout"`
	Cardinal struct {
		Namespace string `toml:"namespace""`
	}
	EVM struct {
		DAAuthToken   string `toml:"da_auth_token"`
		DABaseURL     string `toml:"da_base_url"`
		DANamespaceID string `toml:"da_namespace_id"`
		ChainID       string `toml:"chain_id"`
		KeyMnemonic   string `toml:"key_mnemonic"`
		FaucetAddr    string `toml:"faucet_addr"`
		BlockTime     int    `toml:"block_time"`
	}
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
	cfg := Config{}
	file, err := os.Open(filename)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	if err = toml.NewDecoder(file).Decode(&cfg); err != nil {
		return cfg, err
	}
	log.Debug().Msgf("successfully loaded config from %q", filename)
	cfg.RootDir, _ = filepath.Split(filename)
	return cfg, nil
}
