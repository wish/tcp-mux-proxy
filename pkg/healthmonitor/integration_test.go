package healthmonitor

import (
	"context"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSystemBasic(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	configLocation := "config.sample.yaml"
	config, err := ParseConfig(configLocation)
	if err != nil {
		log.Fatalf("Could not parse config: %v\n", err)
	}
	rand.Seed(time.Now().UnixNano())

	// Initialize proxy and health monitor
	proxy := NewProxyServer(&config)
	healthMonitor := NewHealthMonitor(&config, proxy)
	go MetricsServer(config.Proxy.MetricsPort)

	for _, backend := range config.Backend {
		go runMockDownstream(":" + strconv.Itoa(backend.Port))
		go func(port int) {
			time.Sleep(time.Second * 25)
			runMockDownstream(":" + strconv.Itoa(port))
		}(backend.Port)
	}

	// we can launch the health checks before starting the proxy server
	for id := range config.Backend {
		go healthMonitor.ConfirmHealth(uint16(id))
	}

	go runMockUpstream("http://localhost" + config.Proxy.Bind)
	go func() {
		time.Sleep(time.Second * 30)
		runMockUpstream("http://localhost" + config.Proxy.Bind)
	}()

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

func runMockUpstream(targetURL string) {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	time.Sleep(time.Second * 2)
	var count int64
	var aCount int64
	var wg sync.WaitGroup
	tr := &http.Transport{
		MaxIdleConns: 10,
	}
	client := http.Client{Timeout: time.Second * 5, Transport: tr}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 100; j++ {
				time.Sleep(time.Microsecond * time.Duration(rand.Float32()*100))
				req, err := http.NewRequest("GET", targetURL, nil)
				req.Close = true
				response, err := client.Do(req)
				if err != nil {
					log.Println(err)
				} else {
					body, err := ioutil.ReadAll(response.Body)
					if string(body) == "Hi its Anthony" {
						atomic.AddInt64(&count, 1)
						atomic.AddInt64(&aCount, 1)
					} else {
						atomic.AddInt64(&count, 1)
					}
					if err != nil {
						log.Println(err)
					}
					response.Body.Close()
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	log.Printf("\tcount: %v\n", atomic.LoadInt64(&count))
	log.Printf("\taCount: %v\n", atomic.LoadInt64(&aCount))
}

func runMockDownstream(bind string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		p := rand.Float32()
		if p > 0.999 {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hi its Anthony"))
		}
	})

	server := http.Server{
		Addr:    bind,
		Handler: mux,
	}

	go func() {
		time.Sleep(time.Second * 20)
		server.Shutdown(context.Background())
	}()
	server.ListenAndServe()
}
