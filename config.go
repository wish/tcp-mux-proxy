package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
)

// Config contains the options you can set for the proxy
type Config struct {
	Proxy struct {
		Bind              string        `yaml:"bind"`
		MetricsPort       string        `yaml:"metrics_server_port"`
		MaxConn           int           `yaml:"max_conn"`
		MinAlive          int           `yaml:"min_alive"`
		RecoverySleepTime time.Duration `yaml:"recovery_sleep_time"`
	} `yaml:"proxy"`
	Backend []BackendPort `yaml:"backend"`
}

// BackendPort contains the options you can set for a backend server
type BackendPort struct {
	Name                string        `yaml:"name"`
	Host                string        `yaml:"host"`
	Port                int           `yaml:"port"`
	HealthCheckEndpoint string        `yaml:"health_check_endpoint"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	URL                 *url.URL
}

// ParseConfig parses the configuration file
func ParseConfig(configLocation string) (Config, error) {
	var config Config

	filename, err := filepath.Abs(configLocation)
	if err != nil {
		return Config{}, fmt.Errorf("Invalid file path: %v", err)
	}

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("Error reading config file: %v", err)
	}

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return Config{}, fmt.Errorf("Invalid yaml file: %v", err)
	}

	for i, backend := range config.Backend {
		// convert to type url.URL
		urlString := backend.Host + ":" + strconv.Itoa(backend.Port)
		config.Backend[i].URL, err = url.Parse(urlString)
		if err != nil {
			return Config{}, fmt.Errorf("Invalid URL: %v", err)
		}
	}
	return config, nil
}
