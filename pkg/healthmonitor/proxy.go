package healthmonitor

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/wish/tcp-mux-proxy/pkg/loadbalancer"
)

// ProxyServer encapsulates the server and config for the proxy
type ProxyServer struct {
	server              *http.Server
	ph                  proxyHandler
	bind                string
	shutdownInProgress  uint32
	metrics             *ProxyServerMetrics
	lastStateChangeTime time.Time
	hostnameLabel       prometheus.Labels
	firstStart          bool
}

// NewProxyServer builds a proxy server and returns it
func NewProxyServer(config *Config) *ProxyServer {
	proxyServer := &ProxyServer{
		ph: proxyHandler{
			lb:       loadbalancer.NewPowerOfTwoLoadBalancer(uint16(len(config.Backend))),
			maxConn:  uint32(config.Proxy.MaxConn),
			backends: config.Backend,
			metrics:  NewProxyHandlerMetrics(),
			hostname: config.Backend[0].Host,
		},
		bind:                config.Proxy.Bind,
		shutdownInProgress:  0,
		metrics:             NewProxyServerMetrics(),
		hostnameLabel:       prometheus.Labels{"hostname": config.Backend[0].Host},
		firstStart:          true,
		lastStateChangeTime: time.Now(),
	}

	proxies := make([]*httputil.ReverseProxy, len(config.Backend))
	for i, portConfig := range config.Backend {
		proxies[i] = httputil.NewSingleHostReverseProxy(portConfig.URL)
		proxies[i].Transport = &proxyTransport{id: uint16(i), ph: &proxyServer.ph}
	}
	proxyServer.ph.proxies = proxies
	return proxyServer
}

func (proxyServer *ProxyServer) resetTimer() float64 {
	tmp := proxyServer.lastStateChangeTime
	proxyServer.lastStateChangeTime = time.Now()
	return time.Since(tmp).Seconds()
}

// IsInShutdown returns true if the server is in shutdown, else returns false
func (proxyServer *ProxyServer) IsInShutdown() bool {
	return atomic.LoadUint32(&proxyServer.shutdownInProgress) == 1
}

func (proxyServer *ProxyServer) stop() {
	// this is necessary since stop can also be called from start if ListenAndServe gets an error
	if atomic.CompareAndSwapUint32(&proxyServer.shutdownInProgress, uint32(0), uint32(1)) {
		// TODO make this context cancelable
		if err := proxyServer.server.Shutdown(context.Background()); err != nil {
			// what handling should we have here
		}
		proxyServer.shutdownInProgress = 0
	}
}

// Start starts the proxy server
func (proxyServer *ProxyServer) Start() error {
	// at this point proxyHandler.curConn should be zero after shutdown
	mux := http.NewServeMux()
	mux.Handle("/status/", &statusHandler{})
	mux.Handle("/", &proxyServer.ph)

	proxyServer.server = &http.Server{
		Addr:         proxyServer.bind,
		Handler:      mux,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// we do not want to make an observation of time unhealthy upon the first start
	if proxyServer.firstStart {
		proxyServer.resetTimer()
		proxyServer.firstStart = false
	} else {
		proxyServer.metrics.timeUnhealthy.With(proxyServer.hostnameLabel).Observe(proxyServer.resetTimer())
	}

	log.Println("Starting proxy server")
	defer log.Println("Proxy server has shut down")
	err := proxyServer.server.ListenAndServe()
	proxyServer.metrics.timeHealthy.With(proxyServer.hostnameLabel).Observe(proxyServer.resetTimer())

	if err != http.ErrServerClosed {
		proxyServer.stop()
		return err
	}
	return nil
}

type statusHandler struct{}

func (sh *statusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// haproxy docs say health checks consist of estabilishing tcp connection
	// if server is in shutdown, this will fail, otherwise it will succeed
	// this will only be needed if using httpchk
	// http:// cbonte.github.io/haproxy-dconv/2.0/configuration.html
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

type proxyHandler struct {
	lb       loadbalancer.LoadBalancer
	backends []BackendPort
	maxConn  uint32
	curConn  uint32
	client   http.Client
	metrics  *ProxyHandlerMetrics
	hostname string
	proxies  []*httputil.ReverseProxy
}

func (ph *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tStart := time.Now()
	for {
		localCurConn := atomic.LoadUint32(&ph.curConn)
		if localCurConn >= ph.maxConn {
			// refuse the connection
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if atomic.CompareAndSwapUint32(&ph.curConn, localCurConn, localCurConn+1) {
			break
		}
	}

	id := ph.lb.GetDownstream()
	ph.lb.IncConn(id)

	//proxy := httputil.NewSingleHostReverseProxy(ph.backends[id].URL)
	serveTimeNS := time.Since(tStart).Nanoseconds()
	ph.proxies[id].ServeHTTP(w, r)
	ph.metrics.handleTimeNS.With(prometheus.Labels{"hostname": ph.hostname}).Observe(float64(serveTimeNS))
}

type proxyTransport struct {
	ph *proxyHandler
	id uint16
}

func (pt *proxyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	pt.ph.metrics.httpRequests.With(prometheus.Labels{"hostname": pt.ph.hostname}).Inc()
	pt.ph.metrics.numActiveConnections.With(prometheus.Labels{"hostname": pt.ph.hostname, "port": strconv.Itoa(pt.ph.backends[pt.id].Port)}).Inc()
	response, err := http.DefaultTransport.RoundTrip(request)

	defer func() {
		pt.ph.lb.DecConn(pt.id)
		atomic.AddUint32(&pt.ph.curConn, ^uint32(0))
	}()

	if err != nil {
		return nil, err
	}

	pt.ph.metrics.numActiveConnections.With(prometheus.Labels{"hostname": pt.ph.hostname, "port": strconv.Itoa(pt.ph.backends[pt.id].Port)}).Dec()
	pt.ph.metrics.httpResponses.With(prometheus.Labels{"hostname": pt.ph.hostname, "code": fmt.Sprintf("%vxx", response.StatusCode/100)}).Inc()
	return response, err
}
