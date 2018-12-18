// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import "net/http"
import log "github.com/sirupsen/logrus"

// exposes and endpoint for metrics requests. we can keep adding the routes fro different messages
func RunMetricsServices(address string) {
	http.HandleFunc("/heartbeat", heartbeatHandler)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.WithError(err).Warningf("Failed to start the metrics service. Reason: %s", err.Error())
	}
	log.Infof("metrics service started successfully and available from %s", address)
}
