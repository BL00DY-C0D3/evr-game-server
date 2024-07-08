package wrapper

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type APICachingProxy struct {
	sync.RWMutex
	cachedData string
	originPort int
	listenPort int
	frequency  int
	logger     *log.Logger
	limiter    *rate.Limiter
	client     *http.Client
}

func NewAPICachingProxy(originPort, proxyPort, frequency *int, loggerFlags int) *APICachingProxy {

	return &APICachingProxy{
		cachedData: "",
		originPort: *originPort,
		listenPort: *proxyPort,
		frequency:  *frequency,
		logger:     log.New(os.Stderr, "[api-proxy] ", loggerFlags),
		limiter:    rate.NewLimiter(rate.Limit(30), 1),
		client: &http.Client{
			Transport: &http.Transport{
				MaxConnsPerHost: 1,
				MaxIdleConns:    1,
				IdleConnTimeout: 30 * time.Second,
			},
		},
	}
}

func (p *APICachingProxy) ListenAndProxy(ctx context.Context) {
	http.HandleFunc("/session", p.handleRequest)
	http.ListenAndServe(fmt.Sprintf(":%d", p.listenPort), nil)
}

// queryAPI queries the API and caches the response.
func (p *APICachingProxy) queryAPI(ctx context.Context) string {
	url := fmt.Sprintf("http://127.0.0.1:%d/session", p.originPort)

	// Create a rate limiter that will allow us to query the API up to 30 times a second.
	// If no client is connected, the rate limiter will block the query.

	// Query the API up to 30 times a second, as long as a client is connected.
	// If no client is connected, the rate limiter will block the query.
	resp, err := http.Get(url)
	if err != nil {
		p.logger.Println("Evr GET Error:", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Println("Evr Read Error:", err)
	}
	resp.Body.Close()
	return string(body)
}

func (p *APICachingProxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	var data string
	if p.limiter.Allow() {
		// Allow the request even if the rate limit is exceeded
		ctx, _ := context.WithTimeout(r.Context(), 1000*time.Millisecond/30-3)
		p.Lock()
		data = p.queryAPI(ctx)
		p.cachedData = data
		p.Unlock()
	} else {
		p.RLock()
		data = p.cachedData
		p.RUnlock()
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, data)
}
