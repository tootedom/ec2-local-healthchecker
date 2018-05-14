// Copyright [2018] [Dominic Tootell]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
