package main

import (
	"flag"
	"log"
	"math/rand"
	"time"
)

func main() {
	// Get the configuration data
	var configLocation = flag.String("c", "config.yaml", "Path to the yaml configuration file")
	flag.Parse()
	config, err := ParseConfig(*configLocation)
	if err != nil {
		log.Fatalf("Could not parse config: %v\n", err)
	}

	rand.Seed(time.Now().UnixNano())

	// Initialize proxy and health monitor
	proxy := NewProxyServer(&config)
	healthMonitor := NewHealthMonitor(&config, proxy)

	go metricsServer(config.Proxy.MetricsPort)

	// we can launch the health checks before starting the proxy server
	for id := range config.Backend {
		go healthMonitor.confirmHealth(uint16(id))
	}

	// Main application loop
	for {
		err := proxy.start()
		if err != nil {
			log.Fatalf("Could not start proxy: %v\n", err)
			panic(err)
		}

		for healthMonitor.isUnhealthy() || proxy.isInShutdown() {
			time.Sleep(config.Proxy.RecoverySleepTime)
		}
	}
}
