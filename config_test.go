package main

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	configLocation := "config.sample.yaml"
	config, err := ParseConfig(configLocation)

	if err != nil {
		t.Error(err)
	}

	fmt.Println(config.Proxy)
	fmt.Println(config.Backend)
	fmt.Println(config.Backend[0].HealthCheckInterval)

	// should probably add some asserts here
}
