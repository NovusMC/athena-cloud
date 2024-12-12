package common

import (
	"errors"
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
)

func ReadConfig[T any](configFile string, defaultConfig T) (*T, error) {
	cfg := defaultConfig

	bytes, err := os.ReadFile(configFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {

			bytes, err = yaml.Marshal(cfg)
			if err != nil {
				return nil, fmt.Errorf("error marshaling empty config: %w", err)
			}

			err = os.WriteFile(configFile, bytes, 0644)
			if err != nil {
				return nil, fmt.Errorf("error writing config file: %w", err)
			}

			return &cfg, nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &cfg, nil
}
