package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	FirecrackerBin string `json:"firecrackerBin"`
	KernelPath     string `json:"kernelPath"`
	BaseRootfsPath string `json:"baseRootfsPath"`
	EnvdPath       string `json:"envdPath"`
	SnapshotDir    string `json:"snapshotDir"`
}

func Load(path string) (*Config, error) {
	config, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading config file with path : %s and Error : %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("Error in Unmarshal : %w", err)
	}
	return &cfg, nil
}
