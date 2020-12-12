package metrics

import (
	"bytes"
	"crypto/ecdsa"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"github.com/OneOfOne/go-utils/memory"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/xid"
	"github.com/singnet/snet-daemon/authutils"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"
)

const   MeteringPrefix  = "_usage"
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
func Publish(payload interface{}, serviceUrl string,commonStats *CommonStats) bool {
	jsonBytes, err := ConvertStructToJSON(payload)
	if err != nil {
		return false
	}
	status := publishJson(jsonBytes, serviceUrl, true,commonStats)
	if !status {
		log.WithField("payload", string(jsonBytes)).WithField("url", serviceUrl).Warning("Unable to publish metrics")
	}
	return status
}

// Publish the json on the service end point, retry will be set to false when trying to re publish the payload
func publishJson(json []byte, serviceURL string, reTry bool,commonStats *CommonStats) bool {
	response, err := sendRequest(json, serviceURL,commonStats)
	if err != nil {
		log.WithError(err)
	} else {
		status, reRegister := checkForSuccessfulResponse(response)
		if reRegister && reTry {
			//if Daemon was registered successfully , retry to publish the payload
			status = publishJson(json, serviceURL, false,commonStats)
		}
		return status
	}
	return false
}

//Set all the headers before publishing
func sendRequest(json []byte, serviceURL string,commonStats *CommonStats ) (*http.Response, error) {
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
	SignMessageForMetering(req,commonStats)

	return client.Do(req)

}

func SignMessageForMetering(req *http.Request, commonStats *CommonStats) () {

	privateKey, err := getPrivateKeyForMetering()
	if err != nil {
		log.Error(err)
		return
	}
	currentBlock, err := authutils.CurrentBlock();
	if err != nil {
		log.Error(err)
		return
	}

	signature := signForMeteringValidation(privateKey, currentBlock, MeteringPrefix, commonStats)

	req.Header.Set("X-username", commonStats.UserName)
	req.Header.Set("X-Organizationid", commonStats.OrganizationID)
	req.Header.Set("X-Groupid", commonStats.GroupID)
	req.Header.Set("X-Serviceid", commonStats.ServiceID)
	req.Header.Set("X-Currentblocknumber", currentBlock.String())
	req.Header.Set("X-Signature", b64.StdEncoding.EncodeToString(signature))

}

func getPrivateKeyForMetering()  (privateKey *ecdsa.PrivateKey,err error) {
	if privateKeyString := config.GetString(config.PvtKeyForMetering); privateKeyString != "" {
		privateKey, err = crypto.HexToECDSA(privateKeyString)
		if err != nil {
			return nil, err
		}
		log.WithField("public key",crypto.PubkeyToAddress(privateKey.PublicKey).String())
	}

	return
}

func signForMeteringValidation(privateKey *ecdsa.PrivateKey, currentBlock *big.Int, prefix string,commonStats *CommonStats) []byte {
	message := bytes.Join([][]byte{
		[]byte(prefix),
		[]byte(commonStats.UserName),
		[]byte(commonStats.OrganizationID),
		[]byte(commonStats.ServiceID),
		[]byte(commonStats.GroupID),

		common.BigToHash(currentBlock).Bytes(),
	}, nil)

	return authutils.GetSignature(message, privateKey)
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

		return false, false
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
		log.Infof("Unable to retrieve Token from Body , : %s ", err.Error())
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
