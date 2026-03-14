// config/loader.go
package config

import (
	"encoding/json"
	"os"

	"github.com/tailscale/hujson"
)

func LoadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cleaned, err := hujson.Standardize(file)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(cleaned, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
