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
		status:            newGaugeMetric("tcp_mux_proxy_status", "Current health status of this server (1 = UP, 0 = DOWN)", []string{"server"}),
		numUnhealthyPorts: newGaugeMetric("tcp_mux_proxy_unhealthy_ports", "Current number of unhealthy ports on this server", []string{"server"}),
	}
}

// ProxyServerMetrics contains the proxy server metrics
type ProxyServerMetrics struct {
	//todo config has no reasonable backend name to use here
	timeUnhealthy *prometheus.SummaryVec
	timeHealthy   *prometheus.SummaryVec
}

// NewProxyServerMetrics creates an instance of ProxyServerMetrics
func NewProxyServerMetrics() *ProxyServerMetrics {
	return &ProxyServerMetrics{
		timeUnhealthy: newSummaryMetric("tcp_mux_proxy_continuous_time_unhealthy_seconds", "Length of time for server to come back up", []string{"server"}),
		timeHealthy:   newSummaryMetric("tcp_mux_proxy_continuous_time_healthy_seconds", "Length of time between successive surver shutdowns", []string{"server"}),
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
		httpResponses:        newCounterMetric("tcp_mux_proxy_http_responses_total", "Total of HTTP responses.", []string{"server", "code"}),
		httpRequests:         newCounterMetric("tcp_mux_proxy_http_requests_total", "Total of HTTP requests.", []string{"server"}),
		numActiveConnections: newGaugeMetric("tcp_mux_proxy_port_active_connections", "Current number of active connections for a downstream", []string{"backend"}),
		handleTimeNS:         newSummaryMetric("tcp_mux_proxy_handling_time_ns", "Time in ns to verify num connections is below limit and choose a downstream", []string{"server"}),
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
