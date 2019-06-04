package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/wish/tcp-mux-proxy/pkg/healthmonitor"
)

func main() {
	// Get the configuration data
	var configLocation = flag.String("c", "config.yaml", "Path to the yaml configuration file")
	flag.Parse()
	config, err := healthmonitor.ParseConfig(*configLocation)
	if err != nil {
		log.Fatalf("Could not parse config: %v\n", err)
	}

	rand.Seed(time.Now().UnixNano())

	// Initialize proxy and health monitor
	proxy := healthmonitor.NewProxyServer(&config)
	healthMonitor := healthmonitor.NewHealthMonitor(&config, proxy)

	go healthmonitor.MetricsServer(config.Proxy.MetricsPort)

	// we can launch the health checks before starting the proxy server
	for id := range config.Backend {
		go healthMonitor.ConfirmHealth(uint16(id))
	}

	// Main application loop
	for {
		err := proxy.Start()
		if err != nil {
			log.Fatalf("Could not start proxy: %v\n", err)
			panic(err)
		}

		for healthMonitor.IsUnhealthy() || proxy.IsInShutdown() {
			time.Sleep(config.Proxy.RecoverySleepTime)
		}
	}
}
