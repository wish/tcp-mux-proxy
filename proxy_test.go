package main

import (
	"testing"
	"time"
)

func TestBasicProxyServer(t *testing.T) {
	configLocation := "config.sample.yaml"
	config, err := ParseConfig(configLocation)
	if err != nil {
		t.Error(err)
	}

	proxy := NewProxyServer(&config)
	go proxy.start()
	time.Sleep(time.Second * 10)
	for i := 0; i < 5; i++ {
		go proxy.stop()
	}
}
