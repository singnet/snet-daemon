package httphandler

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
)

type httpHandler struct {
	bp                  blockchain.Processor 
	passthroughEnabled  bool
	passthroughEndpoint string
}

func NewHTTPHandler(blockProc blockchain.Processor) http.Handler {
	return httpHandler{
		bp:                  blockProc, 
		passthroughEnabled:  config.GetBool(config.PassthroughEnabledKey),
		passthroughEndpoint: config.GetString(config.PassthroughEndpointKey),
	}
}

func (h httpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if h.passthroughEnabled {
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
			if body, err := ioutil.ReadAll(resp2.Body); err == nil {
				resp.Write(body)
			} else {
				http.Error(resp, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if body, err := ioutil.ReadAll(req.Body); err == nil {
			resp.Write(body)
		} else {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
	} 
}
