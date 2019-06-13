package healthmonitor

import (
	"io"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// HealthMonitor is responsible for monitoring and
// reporting the health of the downstream ports
type HealthMonitor struct {
	numUnhealthy uint32
	threshold    uint32
	client       http.Client
	proxy        *ProxyServer
	backends     []BackendPort
	metrics      HealthMonitorMetrics
	serverLabel  prometheus.Labels
}

// NewHealthMonitor makes a HealthMonitor and returns it
func NewHealthMonitor(config *Config, proxy *ProxyServer) *HealthMonitor {
	serverLabel := prometheus.Labels{"server": config.Proxy.Name}
	metrics := NewHealthMonitorMetrics()
	metrics.status.With(serverLabel).Inc()

	return &HealthMonitor{
		numUnhealthy: 0,
		threshold:    uint32(len(config.Backend) - config.Proxy.MinAlive), //should add an assert that this is greater than or equal to zero
		proxy:        proxy,
		backends:     config.Backend,
		client:       http.Client{Timeout: time.Second},
		metrics:      metrics,
		serverLabel:  serverLabel,
	}
}

// IsUnhealthy returns true if the server has more unhealthy downstreams
// than the threshold value
func (hm *HealthMonitor) IsUnhealthy() bool {
	return atomic.LoadUint32(&hm.numUnhealthy) >= hm.threshold
}

// ConfirmHealth starts the health check loop
func (hm *HealthMonitor) ConfirmHealth(id uint16) {
	for {
		if !hm.checkHealth(id) {
			hm.incUnhealthy(id)
			hm.recoverHealth(id)
		}
		time.Sleep(hm.backends[id].HealthCheckInterval)
	}
}

func (hm *HealthMonitor) recoverHealth(id uint16) {
	for {
		if hm.checkHealth(id) {
			hm.decUnhealthy(id)
			hm.ConfirmHealth(id)
		}
		time.Sleep(hm.backends[id].HealthCheckInterval)
	}
}

func (hm *HealthMonitor) incUnhealthy(id uint16) {
	if atomic.AddUint32(&hm.numUnhealthy, uint32(1)) == hm.threshold {
		// want to execute this right away
		hm.proxy.stop()
		hm.metrics.status.With(hm.serverLabel).Dec()
	}
	hm.proxy.ph.lb.MarkUnhealthy(id)
	hm.metrics.numUnhealthyPorts.With(hm.serverLabel).Inc()
}

func (hm *HealthMonitor) decUnhealthy(id uint16) {
	if atomic.AddUint32(&hm.numUnhealthy, ^uint32(0)) == hm.threshold-1 {
		hm.metrics.status.With(hm.serverLabel).Inc()
	}
	hm.proxy.ph.lb.MarkHealthy(id)
	hm.metrics.numUnhealthyPorts.With(hm.serverLabel).Dec()
}

func (hm *HealthMonitor) checkHealth(id uint16) bool {
	endpoint := hm.backends[id].URL.String() + hm.backends[id].HealthCheckEndpoint
	req, err := http.NewRequest("GET", endpoint, nil)
	req.Close = true
	response, err := hm.client.Do(req)
	if err != nil {
		return false
	}

	responseVal := (response.StatusCode - 400) / 100
	if responseVal >= 0 {
		return false
	}

	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()

	return true
}
