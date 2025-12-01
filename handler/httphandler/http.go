package httphandler

import (
	"io"
	"log"
	"net/http"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/ratelimit"
	"golang.org/x/time/rate"

	"github.com/singnet/snet-daemon/v6/config"
)

type httpHandler struct {
	passthroughEnabled  bool
	passthroughEndpoint string
	rateLimiter         rate.Limiter
}

func NewHTTPHandler(blockProc blockchain.Processor) http.Handler {
	return &httpHandler{
		passthroughEnabled:  config.GetBool(config.PassthroughEnabledKey),
		passthroughEndpoint: config.GetString(config.ServiceEndpointKey),
		rateLimiter:         *ratelimit.NewRateLimiter(),
	}
}

func (h *httpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	log.Printf("ServeHTTP: %#v \n", req)
	if h.passthroughEnabled {
		if !h.rateLimiter.Allow() {
			http.Error(resp, http.StatusText(429), http.StatusTooManyRequests)
			return
		}
		req2, err := http.NewRequest(req.Method, h.passthroughEndpoint, req.Body)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
		req2.Header = req.Header
		if resp2, err := http.DefaultClient.Do(req2); err == nil {
			for k, l := range resp2.Header {
				for _, v := range l {
					resp.Header().Add(k, v)
				}
			}
			if body, err := io.ReadAll(resp2.Body); err == nil {
				_, err := resp.Write(body)
				if err != nil {
					return
				}
			} else {
				http.Error(resp, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if body, err := io.ReadAll(req.Body); err == nil {
			_, err := resp.Write(body)
			if err != nil {
				return
			}
		} else {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
