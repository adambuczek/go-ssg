// Package config loads and validates the site configuration.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Src            string
	Dist           string
	Layouts        string
	Assets         string
	PollingTimeout time.Duration
	Port           string
}

const (
	defaultPollingTimeout = 1 * time.Second
)

func LoadConfig() (Config, error) {
	path := "config.yaml"
	file, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config file: %v", err)
	}
	var output Config
	err = yaml.Unmarshal(file, &output)
	if err != nil {
		return Config{}, err
	}
	if output.Src == "" {
		return Config{}, fmt.Errorf("config: %s is required", "src")
	}
	if output.Dist == "" {
		return Config{}, fmt.Errorf("config: %s is required", "dist")
	}
	if output.Layouts == "" {
		return Config{}, fmt.Errorf("config: %s is required", "layouts")
	}
	if output.Assets == "" {
		return Config{}, fmt.Errorf("config: %s is required", "assets")
	}
	if output.PollingTimeout == 0 {
		output.PollingTimeout = defaultPollingTimeout
	}
	if output.Port == "" {
		output.Port = "8080"
	}
	return output, nil
}
