package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/OneOfOne/go-utils/memory"
	"github.com/rs/xid"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"io/ioutil"
	"net/http"
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
	log.WithField("payload", string(b)).Debug("payload")
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

// Publish the json on the service end point
func publishJson(json []byte, serviceURL string) bool {
	response, err := sendRequest(json, serviceURL)
	if err != nil {
		log.WithError(err)
	} else {
		status, retry := checkForSuccessfulResponse(response)
		if retry {
			//if Daemon was registered successfully , retry to publish the payload
			status = rePublishJson(json, serviceURL)
		}
		return status
	}
	return false
}

// Re Publish the json on the service end point
func rePublishJson(json []byte, serviceURL string) bool {
	response, err := sendRequest(json, serviceURL)
	if err != nil {
		log.WithError(err).Warningf("%v", response)
		return false
	}
	log.Debugf("Metrics republished with status code : %d ", response.StatusCode)
	status, _ := checkForSuccessfulResponse(response)
	return status
}

//Set all the headers before publishing
func sendRequest(json []byte, serviceURL string) (*http.Response, error) {
	req, err := http.NewRequest("POST", serviceURL, bytes.NewBuffer(json))
	if err != nil {
		log.WithField("serviceURL", serviceURL).WithError(err).Warningf("Unable to create service request to publish stats")
		return nil, err
	}
	// sending the post request
	client := &http.Client{}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Daemonid", GetDaemonID())
	req.Header.Set("X-Token", daemonAuthorizationToken)
	return client.Do(req)

}

//Check if the response received was proper
func checkForSuccessfulResponse(response *http.Response) (status bool, retry bool) {
	if response == nil {
		log.Warningf("Empty response received.")
		return false, false
	}
	if response.StatusCode != http.StatusOK {
		log.Warningf("Service call failed with status code : %d ", response.StatusCode)
		//if response returned was forbidden error , then re register Daemon with fresh token and submit the request / response
		//again ONLY if the Daemon was registered successfully
		status = RegisterDaemon(config.GetString(config.MonitoringServiceEndpoint) + "/register")
		return false, status
	} //close the body
	log.Debugf("Metrics posted successfully with status code : %d ", response.StatusCode)
	defer response.Body.Close()
	return true, false
}

//Check if the response received was proper
func getTokenFromResponse(response *http.Response) (string, bool) {
	if response == nil {
		log.Warningf("Empty response received.")
		return "", false
	}
	if response.StatusCode != http.StatusOK {
		log.Warningf("Service call failed with status code : %d ", response.StatusCode)
		return "", false
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Infof("Unable to retrieve Token from Body , : %f ", err.Error())
		return "", false
	}
	var data TokenGenerated
	json.Unmarshal(body, &data)
	//close the body
	defer response.Body.Close()
	return data.Data.Token, true
}

//Generic utility to determine the size of the srtuct passed
func GetSize(v interface{}) uint64 {
	return memory.Sizeof(v)
}

// returns the epoch UTC timestamp from the current system time
func getEpochTime() int64 {
	return time.Now().UTC().Unix()
}
