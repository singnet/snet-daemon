// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// Package metrics for monitoring and reporting the daemon metrics
package metrics

import (
	context2 "context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"net/http"
	"strconv"
	"strings"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/training"
	"github.com/singnet/snet-daemon/v6/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"golang.org/x/net/context"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Status enum
type Status int

const (
	Offline  Status = 0 // Returns if none of the services are online
	Online   Status = 1 // Returns if any of the services is online
	Warning  Status = 2 // if the daemon has issues in extracting the service state
	Critical Status = 3 // if the daemon main thread killed or any other critical issues
)

type StorageClientCert struct {
	ValidFrom string `json:"validFrom"`
	ValidTill string `json:"validTill"`
}

// DaemonHeartbeat data model. Service Status JSON object Array marshalled to a string
type DaemonHeartbeat struct {
	DaemonID                 string                                     `json:"daemonID"`
	Timestamp                string                                     `json:"timestamp"`
	Status                   string                                     `json:"status"`
	ServiceHeartbeat         string                                     `json:"serviceheartbeat"`
	DaemonVersion            string                                     `json:"daemonVersion"`
	DynamicPricing           map[string]string                          `json:"-"`
	BlockchainEnabled        bool                                       `json:"blockchainEnabled"`
	BlockchainNetwork        string                                     `json:"blockchainNetwork"`
	StorageClientCertDetails StorageClientCert                          `json:"storageClientCertDetails"`
	CurrentBlock             func() (*big.Int, error)                   `json:"-"`
	TrainingMetadata         func() (*training.TrainingMetadata, error) `json:"-"`
	TrainingMetadataData     *training.TrainingMetadata                 `json:"trainingMetadata,omitempty"`
}

func (service *DaemonHeartbeat) List(ctx context2.Context, request *grpc_health_v1.HealthListRequest) (*grpc_health_v1.HealthListResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}

// Converts the enum index into enum names
func (state Status) String() string {
	// declare an array of strings. operator counts how many items in the array (4)
	listStatus := [...]string{"Offline", "Online", "Warning", "Critical"}

	// → `state`: It's one of the values of Status constants.
	// prevent panicking in case of `status` is out of range of Status
	if state < Offline || state > Critical {
		return "Unknown"
	}
	// return the status string constant from the array above.
	return listStatus[state]
}

// ValidateHeartbeatConfig validates the heartbeat configurations
func ValidateHeartbeatConfig(heartbeatType, heartbeatEndpoint string) error {
	// initialize the url state to false
	// check if the configured type is not supported
	if heartbeatType != "grpc" && heartbeatType != "http" && heartbeatType != "https" && heartbeatType != "none" && heartbeatType != "" {
		return fmt.Errorf("unrecognized heartbet service type : '%+v'", heartbeatType)
	}

	// if the URLs are empty, or hbtype is None or empty, consider it as not configured
	if heartbeatType == "" || heartbeatType == "none" || heartbeatEndpoint == "" {
		zap.L().Info("heartbeat service endpoint not configured, will be using service endpoint to ping the service")
		return nil
	}

	if !utils.IsURLValid(heartbeatEndpoint) {
		return errors.New("heartbeat endpoint must be a valid URL")
	}
	return nil
}

// getStorageCertificateDetails returns the storage certificate details
func getStorageCertificateDetails() (cert StorageClientCert) {
	cert = StorageClientCert{}
	certificate, err := tls.LoadX509KeyPair(config.GetString(config.PaymentChannelCertPath), config.GetString(config.PaymentChannelKeyPath))
	if err != nil {
		zap.L().Error("unable to load specific SSL X509 keypair for storage certificate", zap.Error(err))
		return
	}
	if len(certificate.Certificate) > 0 {
		parseCertificate, err := x509.ParseCertificate(certificate.Certificate[0])
		if err != nil {
			zap.L().Error("unable to get certificate info", zap.Error(err))
			return
		}
		cert.ValidFrom = fmt.Sprintf("Valid Since: %+v days", parseCertificate.NotBefore.String())
		cert.ValidTill = fmt.Sprintf("Valid Till: %+v days", parseCertificate.NotAfter.String())
	}
	return
}

type HeartStatus struct {
	ServiceID string `json:"serviceID"`
	Status    string `json:"status"`
}

// GetHeartbeat prepares the heartbeat, which includes calling the underlying service Daemon is serving
func GetHeartbeat(serviceEndpoint string, serviceHeartbeatURL string, heartbeatType string, serviceID string, trainingMetadata func() (*training.TrainingMetadata, error), dynamicPricing map[string]string, currentBlock func() (*big.Int, error)) (heartbeat DaemonHeartbeat, err error) {
	heartbeat = DaemonHeartbeat{
		DaemonID:                 GetDaemonID(),
		Timestamp:                strconv.FormatInt(getEpochTime(), 10),
		Status:                   Offline.String(),
		ServiceHeartbeat:         "",
		DaemonVersion:            config.GetVersionTag(),
		DynamicPricing:           dynamicPricing,
		BlockchainEnabled:        config.GetBool(config.BlockchainEnabledKey),
		BlockchainNetwork:        config.GetString(config.BlockChainNetworkSelected),
		StorageClientCertDetails: getStorageCertificateDetails(),
		CurrentBlock:             currentBlock,
		TrainingMetadata:         trainingMetadata,
	}

	if trainingMetadata != nil {
		md, err := trainingMetadata()
		if err != nil {
			md = nil
		}
		heartbeat.TrainingMetadataData = md
	}

	var curResp = &HeartStatus{Status: "NOT_SERVING", ServiceID: serviceID}
	switch heartbeatType {
	case "grpc":
		var response grpc_health_v1.HealthCheckResponse_ServingStatus
		response, err = callGrpcServiceHeartbeat(serviceHeartbeatURL)
		//Standardize this as well on the response being sent
		heartbeat.Status = response.String()
		zap.L().Debug("Get heartbeat", zap.String("serviceHeartbeatURL", serviceHeartbeatURL), zap.String("response", string(response)), zap.Error(err))
	case "http":
		fallthrough
	case "https":
		var serviceHeartbeatBytes []byte
		serviceHeartbeatBytes, err = callHTTPServiceHeartbeat(serviceHeartbeatURL)
		heartbeat.ServiceHeartbeat = string(serviceHeartbeatBytes)
		zap.L().Debug("Get heartbeat", zap.String("serviceHeartbeatURL", serviceHeartbeatURL), zap.String("serviceHeartbeatBytes", string(serviceHeartbeatBytes)), zap.Error(err))
	case "none":
		fallthrough
	case "":
		// trying to ping the service with serviceEndpoint
		err = tcpPingService(serviceEndpoint)
		zap.L().Debug("Get heartbeat [tcpPingService]", zap.String("serviceEndpoint", serviceEndpoint), zap.Error(err))
	}

	if err == nil {
		curResp.Status = "SERVING"
		heartbeat.Status = Online.String()
	}

	if err != nil {
		heartbeat.Status = Offline.String()
		// send the alert if service heartbeat fails
		if config.GetString(config.AlertsEMail) != "" {
			notification := &Notification{
				Recipient: config.GetString(config.AlertsEMail),
				Details:   err.Error(),
				Timestamp: time.Now().String(),
				Message:   "Problem in calling Service Heartbeat endpoint.",
				Component: "Daemon",
				DaemonID:  GetDaemonID(),
				Level:     "ERROR",
			}
			c, err := currentBlock()
			if err != nil {
				zap.L().Error("Error getting current block", zap.Error(err))
			}
			go notification.Send(c)
		}
	}

	marshal, err := json.Marshal(curResp)
	if err != nil {
		return heartbeat, err
	}
	heartbeat.ServiceHeartbeat = string(marshal)
	return heartbeat, err
}

// HeartbeatHandler request handler function: upon request it will hit the service for status and
// wraps the results in daemon's heartbeat
func HeartbeatHandler(rw http.ResponseWriter, trainingMetadata func() (*training.TrainingMetadata, error), dynamicPricing map[string]string, currentBlock func() (*big.Int, error)) {
	// read the heartbeat service type and corresponding URL
	heartbeatType := config.GetString(config.ServiceHeartbeatType)
	serviceURL := config.GetString(config.HeartbeatServiceEndpoint)
	serviceID := config.GetString(config.ServiceId)
	heartbeat, _ := GetHeartbeat(config.GetString(config.ServiceEndpointKey), serviceURL, heartbeatType, serviceID, trainingMetadata, dynamicPricing, currentBlock)
	err := json.NewEncoder(rw).Encode(heartbeat)
	if err != nil {
		zap.L().Info("Failed to write heartbeat message.", zap.Error(err))
	}
}

// Check implements `service Health`.
func (service *DaemonHeartbeat) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {

	heartbeat, err := GetHeartbeat(config.GetString(config.ServiceEndpointKey), config.GetString(config.HeartbeatServiceEndpoint), config.GetString(config.ServiceHeartbeatType),
		config.GetString(config.ServiceId), service.TrainingMetadata, service.DynamicPricing, service.CurrentBlock)

	if err != nil {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_UNKNOWN,
		}, fmt.Errorf("service heartbeat unknown: %w", err)
	}

	if strings.Compare(heartbeat.Status, Online.String()) == 0 {
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
	}

	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN}, errors.New("Service heartbeat unknown: " + heartbeat.Status)
}

// Watch implements `service Watch todo for later`.
func (service *DaemonHeartbeat) Watch(*grpc_health_v1.HealthCheckRequest, grpc_health_v1.Health_WatchServer) error {
	return nil
}

/*
service heartbeat/grpc heartbeat
{"serviceID":"sample1", "status":"SERVING"}

daemon heartbeat
{
  "daemonID": "3a4ebeb75eace1857a9133c7a50bdbb841b35de60f78bc43eafe0d204e523dfe",
  "timestamp": "2018-12-26 22:50:13.4569654 +0000 UTC",
  "status": "Online",
  "serviceheartbeat": "{\"serviceID\":\"sample1\", \"status\":\"SERVING\"}"
}
*/
