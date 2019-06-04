package healthmonitor

import (
	"io"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsServer launches the prometheus metrics server
func MetricsServer(bind string) {
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/status", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"alive":true}`)
	}))

	log.Println("Starting metrics server")
	if err := http.ListenAndServe(bind, nil); err != nil {
		panic(err)
	}
}

// HealthMonitorMetrics contains the health monitor metrics
type HealthMonitorMetrics struct {
	numUnhealthyPorts *prometheus.GaugeVec
	status            *prometheus.GaugeVec
}

// NewHealthMonitorMetrics creates an instance of HealthMonitorMetrics
func NewHealthMonitorMetrics() HealthMonitorMetrics {
	return HealthMonitorMetrics{
		status:            newGaugeMetric("status", "Current health status of this server (1 = UP, 0 = DOWN)", []string{"hostname"}),
		numUnhealthyPorts: newGaugeMetric("unhealthy_ports", "Current number of unhealthy ports on this server", []string{"hostname"}),
	}
}

// ProxyServerMetrics contains the proxy server metrics
type ProxyServerMetrics struct {
	timeUnhealthy *prometheus.SummaryVec
	timeHealthy   *prometheus.SummaryVec
}

// NewProxyServerMetrics creates an instance of ProxyServerMetrics
func NewProxyServerMetrics() *ProxyServerMetrics {
	return &ProxyServerMetrics{
		timeUnhealthy: newSummaryMetric("continuous_time_unhealthy_seconds", "Length of time for server to come back up", []string{"hostname"}),
		timeHealthy:   newSummaryMetric("continuous_time_healthy_seconds", "Length of time between successive surver shutdowns", []string{"hostname"}),
	}
}

// ProxyHandlerMetrics contains the proxy handler metrics
type ProxyHandlerMetrics struct {
	httpResponses        *prometheus.CounterVec
	httpRequests         *prometheus.CounterVec
	numActiveConnections *prometheus.GaugeVec
	handleTimeNS         *prometheus.SummaryVec
}

// NewProxyHandlerMetrics creates an instance of ProxyHandlerMetrics
func NewProxyHandlerMetrics() *ProxyHandlerMetrics {
	return &ProxyHandlerMetrics{
		httpResponses:        newCounterMetric("http_responses_total", "Total of HTTP responses.", []string{"hostname", "code"}),
		httpRequests:         newCounterMetric("http_requests_total", "Total of HTTP requests.", []string{"hostname"}),
		numActiveConnections: newGaugeMetric("port_active_connections", "Current number of active connections for a port", []string{"hostname", "port"}),
		handleTimeNS:         newSummaryMetric("handling_time_ns", "Time in ns to verify num connections is below limit and choose a downstream", []string{"hostname"}),
	}
}

func newGaugeMetric(metricName string, docString string, labels []string) *prometheus.GaugeVec {
	metric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metricName,
			Help: docString,
		},
		labels,
	)
	prometheus.MustRegister(metric)
	return metric
}

func newCounterMetric(metricName string, docString string, labels []string) *prometheus.CounterVec {
	metric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metricName,
			Help: docString,
		},
		labels,
	)
	prometheus.MustRegister(metric)
	return metric
}

func newSummaryMetric(metricName string, docString string, labels []string) *prometheus.SummaryVec {
	metric := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: metricName,
			Help: docString,
		},
		labels,
	)
	prometheus.MustRegister(metric)
	return metric
}
