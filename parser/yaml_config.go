package parser

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type YAMLConfig struct {
	DB struct {
		URI        string `yaml:"uri"`
		Database   string `yaml:"database"`
		Collection string `yaml:"collection"`
	} `yaml:"db"`

	Logic struct {
		DelayBetweenPages int `yaml:"delay_between_pages"`
		ReCrawlInterval    int `yaml:"re_crawl_interval"`
	} `yaml:"logic"`

	Browser struct {
		UseBrowser   bool `yaml:"use_browser"`
		ShowBrowser  bool `yaml:"show_browser"`
		BrowserDebug bool `yaml:"browser_debug"`
	} `yaml:"browser,omitempty"`

	Workers int `yaml:"workers,omitempty"`

	Site string `yaml:"site,omitempty"`
}

func LoadYAMLConfig(configPath string) (*YAMLConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config YAMLConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if config.Logic.DelayBetweenPages <= 0 {
		config.Logic.DelayBetweenPages = 500
	}
	if config.Workers <= 0 {
		config.Workers = 4
	}
	if config.Site == "" {
		config.Site = "both"
	}

	return &config, nil
}

