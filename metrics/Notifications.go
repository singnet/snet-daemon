// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"bytes"
	"github.com/singnet/snet-daemon/config"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// define heartbeat data model. Service Status JSON object Array marshalled to a string
type Notification struct {
	DaemonID  string `json:"daemonID"`
	Timestamp string `json:"timestamp"`
	To        string `json:"to"`
	Message   string `json:"message"`
}

// function for sending an alert to a given endpoint
func (alert *Notification) send() bool {
	url := config.GetString(config.NotificationURL)
	// TODO convert the notification json to string
	var jsonStr = []byte(`{"DaemonID":"eea21ae21"}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Info("Unable to post the metrics - %s", jsonStr)
	}
	defer resp.Body.Close()
	// Read the response
	body, _ := ioutil.ReadAll(resp.Body)

	// TODO parse the notification result. could be a json result
	// convert the result to bool, expectation from service is True/False
	// based on the notification success/failure
	result, _ := strconv.ParseBool(string(body))

	return result
}
