package config

import (
	"io/ioutil"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

type Check struct {
	Threshold int           `yaml:"threshold"`
	Timeout   time.Duration `yaml:"timeout"`
	Endpoint  string        `yaml:"endpoint"`
	Type      string        `yaml:"type"`
	Frequency time.Duration `yaml:"frequency"`
}

type Config struct {
	Frequency   time.Duration    `yaml:"frequency"`
	GracePeriod time.Duration    `yaml:"graceperiod"`
	Checks      map[string]Check `yaml:"checks"`
}

func Load(path string) (*Config, error) {
	filename, _ := filepath.Abs(path)

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := Config{Frequency: time.Second * 10, GracePeriod: time.Minute * 5}

	err = yaml.Unmarshal(yamlFile, &config)

	if err != nil {
		return nil, err
	}

	return &config, nil

}
