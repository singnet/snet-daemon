package escrow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
	"github.com/singnet/snet-daemon/metrics"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

const (

	// EscrowPaymentType each call should have id and nonce of payment channel
	// in metadata.
	FreeCallPaymentType = "free-call"
)

type freeCallPaymentHandler struct {
	service                  FreeCallUserService
	freeCallPaymentValidator *FreeCallPaymentValidator
	orgMetadata              *blockchain.OrganizationMetaData
	serviceMetadata          *blockchain.ServiceMetadata
}

// NewPaymentHandler retuns new MultiPartyEscrow contract payment handler.
func FreeCallPaymentHandler(
	processor *blockchain.Processor, metadata *blockchain.OrganizationMetaData, pServiceMetaData *blockchain.ServiceMetadata) handler.PaymentHandler {
	return &freeCallPaymentHandler{
		orgMetadata:     metadata,
		serviceMetadata: pServiceMetaData,
		freeCallPaymentValidator: NewFreeCallPaymentValidator(processor.CurrentBlock,
			pServiceMetaData.FreeCallSignerAddress()),
	}
}

func (h *freeCallPaymentHandler) Type() (typ string) {
	return FreeCallPaymentType
}

func (h *freeCallPaymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *handler.GrpcError) {
	internalPayment, err := h.getPaymentFromContext(context)
	if err != nil {
		return
	}

	e := h.freeCallPaymentValidator.Validate(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcError(e)
	}

	allowed, errorSeen := h.checkIfFreeCallsAreAllowed(internalPayment.UserId)
	if errorSeen != nil {
		return nil, paymentErrorToGrpcError(errorSeen)
	}
	if !allowed {
		return nil, paymentErrorToGrpcError(fmt.Errorf("free call limit has been exceeded."))
	}

	if err != nil {
		return
	}

	transaction, e := h.service.StartFreeCallUserTransaction(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcError(e)
	}

	return transaction, nil
}

func (h *freeCallPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *FreeCallPayment, err *handler.GrpcError) {

	organizationId := config.GetString(config.OrganizationId)
	serviceId := config.GetString(config.ServiceId)

	userID, err := handler.GetSingleValue(context.MD, handler.FreeCallUserIdHeader)
	if err != nil {
		return
	}

	blockNumber, err := handler.GetBigInt(context.MD, handler.CurrentBlockNumberHeader)
	if err != nil {
		return
	}

	signature, err := handler.GetBytes(context.MD, handler.PaymentChannelSignatureHeader)
	if err != nil {
		return
	}

	return &FreeCallPayment{
		OrganizationId:     organizationId,
		ServiceId:          serviceId,
		UserId:             userID,
		CurrentBlockNumber: blockNumber,
		Signature:          signature,
	}, nil
}

func (h *freeCallPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*freeCallTransaction).Commit())
}

func (h *freeCallPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*freeCallTransaction).Rollback())
}

//todo
func (h *freeCallPaymentHandler) checkIfFreeCallsAreAllowed(username string) (allowed bool, err error) {
	response, err := h.sendRequest(nil, config.GetString(config.FreeCallEndPoint)+"/pricing/usage", username)
	if err != nil {
		return false, err
	}
	//TODO, now get this from store and check
	return h.areFreeCallsExhausted(response)
}

//Set all the headers before publishing
func (h *freeCallPaymentHandler) sendRequest(json []byte, serviceURL string, username string) (*http.Response, error) {
	req, err := http.NewRequest("GET", serviceURL, bytes.NewBuffer(json))
	if err != nil {
		log.WithField("serviceURL", serviceURL).WithError(err).Warningf("Unable to create service request to publish stats")
		return nil, err
	}
	// sending the get request
	q := req.URL.Query()
	orgId := config.GetString(config.OrganizationId)
	serviceId := config.GetString(config.ServiceId)
	q.Add(config.OrganizationId, orgId)
	q.Add(config.ServiceId, serviceId)
	q.Add("username", username)
	metrics.SignMessageForMetering(req,
		&metrics.CommonStats{OrganizationID: orgId, ServiceID: serviceId,
			GroupID: h.orgMetadata.GetGroupIdString(), UserName: username})
	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	req.Header.Set("Content-Type", "application/json")

	return client.Do(req)

}

type FreeCallCheckResponse struct {
	Username       string `json:"username"`
	OrgID          string `json:"org_id"`
	ServiceID      string `json:"service_id"`
	TotalCallsMade int    `json:"total_calls_made"`
}

//Check if the response received was proper
func (h *freeCallPaymentHandler) areFreeCallsExhausted(response *http.Response) (allowed bool, err error) {
	if response == nil {
		log.Warningf("Empty response received.")
		return false, nil
	}
	if response.StatusCode != http.StatusOK {
		log.Warningf("Service call failed with status code : %d ", response.StatusCode)
		return false, nil
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Infof("Unable to retrieve calls allowed from Body , : %f ", err.Error())
		return false, err
	}
	var data FreeCallCheckResponse
	if err = json.Unmarshal(body, &data); err != nil {
		return false, err
	}
	//close the body
	defer response.Body.Close()

	return data.TotalCallsMade < h.serviceMetadata.GetFreeCallsAllowed(), nil
}
