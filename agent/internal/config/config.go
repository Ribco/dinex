package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Listen    string `json:"listen"`
	DataDir   string `json:"data_dir"`
	AuthToken string `json:"auth_token"`
	NodeID    string `json:"node_id"`
	NodeName  string `json:"node_name"`
	Version   string `json:"version"`
}

func Load() (*Config, error) {
	path := os.Getenv("DINEX_CONFIG")

	if path == "" {
		path = "/etc/dinex/agent.json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := Default()

			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return nil, err
			}

			encoded, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return nil, err
			}

			if err := os.WriteFile(path, encoded, 0600); err != nil {
				return nil, err
			}

			return cfg, nil
		}

		return nil, err
	}

	var cfg Config

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Listen == "" {
		cfg.Listen = "0.0.0.0:8080"
	}

	if cfg.DataDir == "" {
		cfg.DataDir = "/var/lib/dinex"
	}

	if cfg.Version == "" {
		cfg.Version = "0.1.0"
	}

	return &cfg, nil
}

func Default() *Config {
	return &Config{
		Listen:   "0.0.0.0:8080",
		DataDir:  "/var/lib/dinex",
		NodeID:   "development",
		NodeName: "Development Node",
		Version:  "0.1.0",
	}
}
