package httphandler

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/ratelimit"
	"golang.org/x/time/rate"

	"github.com/singnet/snet-daemon/v6/config"
)

type httpHandler struct {
	passthroughEnabled  bool
	passthroughEndpoint string
	rateLimiter         rate.Limiter
	timeout             time.Duration
	client              *http.Client
}

func NewHTTPHandler(blockProc blockchain.Processor) *httpHandler {
	transport := http.DefaultTransport
	if standard, ok := transport.(*http.Transport); ok {
		transport = standard.Clone()
	}
	return &httpHandler{
		passthroughEnabled:  config.GetBool(config.PassthroughEnabledKey),
		passthroughEndpoint: config.GetString(config.ServiceEndpointKey),
		rateLimiter:         *ratelimit.NewRateLimiter(),
		timeout:             config.GetDuration(config.ServiceTimeout),
		client:              &http.Client{Transport: transport},
	}
}

func (h *httpHandler) Close() { h.client.CloseIdleConnections() }

func (h *httpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	log.Printf("ServeHTTP: %#v \n", req)
	if h.passthroughEnabled {
		if h.timeout > 0 {
			ctx, cancel := context.WithTimeout(req.Context(), h.timeout)
			defer cancel()
			req = req.WithContext(ctx)
		}
		if !h.rateLimiter.Allow() {
			http.Error(resp, http.StatusText(429), http.StatusTooManyRequests)
			return
		}
		req2, err := http.NewRequestWithContext(req.Context(), req.Method, h.passthroughEndpoint, req.Body)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
		req2.Header = req.Header
		if resp2, err := h.client.Do(req2); err == nil {
			defer resp2.Body.Close()
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
