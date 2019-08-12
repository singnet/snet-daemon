package escrow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
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
	freeCallPaymentValidator *FreeCallPaymentValidator
}


// NewPaymentHandler retuns new MultiPartyEscrow contract payment handler.
func FreeCallPaymentHandler(
	processor *blockchain.Processor) handler.PaymentHandler {
	return &freeCallPaymentHandler{
		freeCallPaymentValidator: NewFreeCallPaymentValidator(processor.CurrentBlock,
			common.HexToAddress(blockchain.ToChecksumAddress(config.GetString(config.FreeCallSignerAddress)))),
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

	allowed,_ := checkIfFreeCallsAreAllowed(internalPayment.UserId)
	if !allowed {
		return nil,paymentErrorToGrpcError(fmt.Errorf("free call limit has been exceeded."))
	}

	return internalPayment, nil
}

func (h *freeCallPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *FreeCallPayment, err *handler.GrpcError) {

	organizationId := config.GetString(config.OrganizationId)
	serviceId := config.GetString(config.ServiceId)

	userID , err := handler.GetSingleValue(context.MD, handler.FreeCallUserIdHeader)
	if err != nil {
		return
	}

	blockNumber,err := handler.GetBigInt(context.MD,handler.CurrentBlockNumberHeader)
	if err != nil {
		return
	}

	signature,err := handler.GetBytes(context.MD, handler.PaymentChannelSignatureHeader)
	if err != nil {
		return
	}


	return &FreeCallPayment{
		OrganizationId:organizationId,
		ServiceId:serviceId,
		UserId:userID,
		CurrentBlockNumber:blockNumber,
		Signature:          signature,
	}, nil
}

func (h *freeCallPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return nil
}

func (h *freeCallPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return nil
}

func checkIfFreeCallsAreAllowed(username string) (allowed bool, err error) {
	response,err := sendRequest(nil,config.GetString(config.MeteringEndPoint)+"/usage/freecalls",username)
	return checkResponse(response)
}

//Set all the headers before publishing
func sendRequest(json []byte, serviceURL string,username string) (*http.Response, error) {
	req, err := http.NewRequest("GET", serviceURL, bytes.NewBuffer(json))
	if err != nil {
		log.WithField("serviceURL", serviceURL).WithError(err).Warningf("Unable to create service request to publish stats")
		return nil, err
	}
	// sending the get request
	q := req.URL.Query()
	q.Add(config.OrganizationId, config.GetString(config.OrganizationId))
	q.Add(config.ServiceId, config.GetString(config.ServiceId))
	q.Add("username", username)
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
	FreeCallsAllowed      int    `json:"free_calls_allowed"`
}

//Check if the response received was proper
func checkResponse(response *http.Response) (allowed bool,err error) {
	if response == nil {
		log.Warningf("Empty response received.")
		return false , nil
	}
	if response.StatusCode != http.StatusOK {
		log.Warningf("Service call failed with status code : %d ", response.StatusCode)
		return false , nil
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Infof("Unable to retrieve calls allowed from Body , : %f ", err.Error())
		return false , err
	}
	var data FreeCallCheckResponse
	json.Unmarshal(body, &data)
	//close the body
	defer response.Body.Close()

	return data.TotalCallsMade < data.FreeCallsAllowed ,nil
}