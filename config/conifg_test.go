package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func Test_ParseFile(t *testing.T) {
	input := []byte(`checks:
  memcached:
    type: tcp
    timeout: 1s
    endpoint: "localhost:11211"
    threshold: 4
    frequency: 10s
  google:
    type: http
    timeout: 4s
    endpoint: "https://www.google.com"
    threshold: 3
    frequency: 3s`)

	tests := []struct {
		name    string
		wantOut Config
	}{
		{
			name: "test",
			wantOut: Config{
				Checks: map[string]Check{
					"memcached": {Threshold: 4, Timeout: time.Second * 1, Endpoint: "localhost:11211", Type: "tcp", Frequency: time.Second * 10},
					"google":    {Threshold: 3, Timeout: time.Second * 4, Endpoint: "https://www.google.com", Type: "http", Frequency: time.Second * 3},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ParseFile(input)
			require.NoError(t, err)
			require.NotNil(t, actual)
			require.Len(t, actual.Checks, 2)
			assert.Equal(t, tt.wantOut.Checks, actual.Checks)
		})
	}

}

func ParseFile(yamlFile []byte) (Config, error) {
	var f Config
	err := yaml.Unmarshal(yamlFile, &f)
	return f, err
}
