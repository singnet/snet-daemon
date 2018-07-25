package handler

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
	blockchainEnabled   bool
	passthroughEnabled  bool
	passthroughEndpoint string
}

func NewHTTPHandler(blockProc blockchain.Processor) http.Handler {
	return httpHandler{
		bp:                  blockProc,
		blockchainEnabled:   config.GetBool(config.BlockchainEnabledKey),
		passthroughEnabled:  config.GetBool(config.PassthroughEnabledKey),
		passthroughEndpoint: config.GetString(config.PassthroughEndpointKey),
	}
}

func (h httpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var jobAddress, jobSignature string
	var jobAddressBytes, jobSignatureBytes []byte

	if h.blockchainEnabled {
		jobAddress, jobSignature = req.Header.Get(blockchain.JobAddressHeader),
			req.Header.Get(blockchain.JobSignatureHeader)

		// Backward-compatibility for old auth embedded in JSON-RPC request params object
		if jobAddress == "" && jobSignature == "" {
			if bodyBytes, err := ioutil.ReadAll(req.Body); err == nil {
				b := new(interface{})
				json.Unmarshal(bodyBytes, b)
				if bMap, ok := (*b).(map[string]interface{}); ok {
					if p, ok := bMap["params"]; ok {
						if pMap, ok := p.(map[string]interface{}); ok {
							if jA, ok := pMap["job_address"]; ok {
								if jS, ok := pMap["job_signature"]; ok {
									jobAddress, _ = jA.(string)
									jobSignature, _ = jS.(string)
									delete(pMap, "job_address")
									delete(pMap, "job_signature")
								}
							}
							bMap["params"] = pMap
							bodyBytes, _ = json.Marshal(bMap)
						}
					}
				}
				req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		jobAddressBytes, jobSignatureBytes = common.FromHex(jobAddress), common.FromHex(jobSignature)

		if !h.bp.IsValidJobInvocation(jobAddressBytes, jobSignatureBytes) {
			http.Error(resp, "job invocation not valid", http.StatusUnauthorized)
			return
		}
	}

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

	if h.blockchainEnabled {
		h.bp.CompleteJob(jobAddressBytes, jobSignatureBytes)
	}
}
