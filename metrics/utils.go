package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/OneOfOne/go-utils/memory"
	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"net/http"
	"net/url"
	"time"
)

//Get the value of the first Pair
func GetValue(md metadata.MD, key string) string {
	array := md.Get(key)
	if len(array) == 0 {
		return ""
	}
	return array[0]
}

//convert the given struct to its corresponding json.
func ConvertStructToJSON(payLoad interface{}) ([]byte, error) {
	if payLoad == nil {
		return nil, errors.New("empty payload passed")
	}
	b, err := json.Marshal(&payLoad)
	if err != nil {
		log.WithError(err).Warningf("Json conversion error.")
		log.WithField("payLoad", payLoad).Warningf("Unable to derive json from structure passed")
		return nil, err
	}
	return b, nil
}

//Generate a unique global Id
func GenXid() string {
	id := xid.New()
	return id.String()
}

//convert the payload to JSON and publish it to the serviceUrl passed
func Publish(payload interface{}, serviceUrl string) bool {
	jsonBytes, err := ConvertStructToJSON(payload)
	if err != nil {
		return false
	}
	status := publishJson(jsonBytes, serviceUrl)
	if !status {
		log.WithField("payload", string(jsonBytes)).WithField("url", serviceUrl).Warning("Unable to publish metrics")
	}
	return status
}

// isValidUrl tests a string to determine if it is a url or not.
func isValidUrl(urlToTest string) bool {
	_, err := url.ParseRequestURI(urlToTest)
	if err != nil {
		return false
	} else {
		return true
	}
}

// Publish the json on the service end point
func publishJson(json []byte, serviceURL string) bool {
	//prepare the request payload
	req, err := http.NewRequest("POST", serviceURL, bytes.NewBuffer(json))
	if err != nil {
		log.WithField("serviceURL", serviceURL).WithError(err).Warningf("Unable to create service request to publish stats")
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Access-Token", GetDaemonID())
	// sending the post request
	client := &http.Client{}
	response, err := client.Do(req)

	if err != nil {
		log.WithError(err).Warningf("r")
	} else {
		return checkForSuccessfulResponse(response)
	}
	log.WithField("json", json).WithField("url", serviceURL).Warningf("Unable to publish the json to the service ")
	return false
}

//Check if the response received was proper
func checkForSuccessfulResponse(response *http.Response) bool {
	if response == nil {
		log.Warningf("Empty response received.")
		return false
	}
	if response.StatusCode != http.StatusOK {
		log.Warningf("Service call failed with status code : %d ", response.StatusCode)
		return false
	} //close the body
	log.Debugf("Metrics posted successfully with status code : %d ", response.StatusCode)
	defer response.Body.Close()
	return true
}

//Generic utility to determine the size of the srtuct passed
func GetSize(v interface{}) uint64 {
	return memory.Sizeof(v)
}

// returns the epoch UTC timestamp from the current system time
func getEpochTime() int64 {
	return time.Now().UTC().Unix()
}
