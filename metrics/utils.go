package metrics

import (
	"bytes"
	"crypto/ecdsa"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/xid"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

const MeteringPrefix = "_usage"

// GetValue Get the value of the first Pair
func GetValue(md metadata.MD, key string) string {
	array := md.Get(key)
	if len(array) == 0 {
		return ""
	}
	return array[0]
}

// ConvertStructToJSON convert the given struct to its corresponding json.
func ConvertStructToJSON(payload any) ([]byte, error) {
	if payload == nil {
		return nil, errors.New("empty payload passed")
	}
	b, err := json.Marshal(&payload)
	if err != nil {
		zap.L().Warn("json conversion error.", zap.Error(err))
		zap.L().Warn("unable to derive json from structure passed", zap.Any("payload", payload))
		return nil, err
	}
	zap.L().Debug("success json conversion", zap.Any("payload", payload))
	return b, nil
}

// GenXid Generate a unique global Id
func GenXid() string {
	id := xid.New()
	return id.String()
}

// Publish convert the payload to JSON and publish it to the serviceUrl passed
func Publish(payload any, serviceUrl string, commonStats *CommonStats, currentBlock *big.Int) bool {
	jsonBytes, err := ConvertStructToJSON(payload)
	if err != nil {
		return false
	}
	status := publishJson(jsonBytes, serviceUrl, true, commonStats, currentBlock)
	if !status {
		zap.L().Warn("Unable to publish metrics", zap.Any("payload", jsonBytes), zap.Any("url", serviceUrl))
	}
	return status
}

// Publish the JSON on the service end point, retry will be set to false when trying to re publish the payload
func publishJson(json []byte, serviceURL string, reTry bool, commonStats *CommonStats, currentBlock *big.Int) bool {
	response, err := sendRequest(json, serviceURL, commonStats, currentBlock)
	if err != nil {
		zap.L().Error(err.Error())
	} else {
		status, reRegister := checkForSuccessfulResponse(response)
		if reRegister && reTry {
			//if Daemon was registered successfully, retry to publish the payload
			status = publishJson(json, serviceURL, false, commonStats, currentBlock)
		}
		return status
	}
	return false
}

// Set all the headers before publishing
func sendRequest(json []byte, serviceURL string, commonStats *CommonStats, currentBlock *big.Int) (*http.Response, error) {
	req, err := http.NewRequest("POST", serviceURL, bytes.NewBuffer(json))
	if err != nil {
		zap.L().Warn("Unable to create service request to publish stats", zap.Any("serviceURL", serviceURL))
		return nil, err
	}
	// sending the post request
	commonStats.ServiceID = config.GetString(config.ServiceId)
	commonStats.OrganizationID = config.GetString(config.OrganizationId)

	client := &http.Client{}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Daemonid", GetDaemonID())
	req.Header.Set("X-Token", daemonAuthorizationToken)
	SignMessageForMetering(req, commonStats, currentBlock)

	return client.Do(req)
}

func SignMessageForMetering(req *http.Request, commonStats *CommonStats, currentBlock *big.Int) {

	privateKey, err := getPrivateKeyForMetering()
	if err != nil {
		zap.L().Error(err.Error())
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

func getPrivateKeyForMetering() (privateKey *ecdsa.PrivateKey, err error) {
	if privateKeyString := config.GetString(config.PvtKeyForMetering); privateKeyString != "" {
		privateKey, err = crypto.HexToECDSA(privateKeyString)
		if err != nil {
			return nil, err
		}
		zap.L().Info("Get public key for metering", zap.String("public key", crypto.PubkeyToAddress(privateKey.PublicKey).String()))
	}

	return
}

func signForMeteringValidation(privateKey *ecdsa.PrivateKey, currentBlock *big.Int, prefix string, commonStats *CommonStats) []byte {
	message := bytes.Join([][]byte{
		[]byte(prefix),
		[]byte(commonStats.UserName),
		[]byte(commonStats.OrganizationID),
		[]byte(commonStats.ServiceID),
		[]byte(commonStats.GroupID),

		common.BigToHash(currentBlock).Bytes(),
	}, nil)

	return utils.GetSignature(message, privateKey)
}

// Check if the response received was proper
func checkForSuccessfulResponse(response *http.Response) (status bool, retry bool) {
	if response == nil {
		zap.L().Warn("Empty response received.")
		return false, false
	}
	if response.StatusCode != http.StatusOK {
		zap.L().Warn("Service call failed", zap.Int("StatusCode", response.StatusCode))
		//if response returned was forbidden error, then re register Daemon with a fresh token and submit the request / response
		//again ONLY if the Daemon was registered successfully

		return false, false
	} //close the body
	zap.L().Debug("Metrics posted successfully", zap.Int("StatusCode", response.StatusCode))
	defer response.Body.Close()
	return true, false
}

// Check if the response received was proper
func getTokenFromResponse(response *http.Response) (string, bool) {
	if response == nil {
		zap.L().Warn("Empty response received.")
		return "", false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		zap.L().Warn("Service call failed", zap.Int("StatusCode", response.StatusCode))
		return "", false
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		zap.L().Info("Unable to retrieve Token from Body", zap.Error(err))
		return "", false
	}
	var data TokenGenerated
	err = json.Unmarshal(body, &data)
	if err != nil {
		zap.L().Error("Can't unmarshal TokenGenerated", zap.Error(err))
		return "", false
	}
	// close the body
	return data.Data.Token, true
}

//// Generic utility to determine the size of the srtuct passed
//func GetSize(v any) uint64 {
//	return memory.Sizeof(v)
//}

// returns the epoch UTC timestamp from the current system time
func getEpochTime() int64 {
	return time.Now().UTC().Unix()
}
