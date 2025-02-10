package escrow

import (
	"fmt"
	"math/big"

	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"
	"go.uber.org/zap"

	"golang.org/x/net/context"
)

type FreeCallStateService struct {
	orgMetadata       *blockchain.OrganizationMetaData
	serviceMetadata   *blockchain.ServiceMetadata
	freeCallService   FreeCallUserService
	freeCallValidator *FreeCallPaymentValidator
}

func (service *FreeCallStateService) mustEmbedUnimplementedFreeCallStateServiceServer() {
	//TODO implement me
	panic("implement me")
}

func NewFreeCallStateService(orgMetadata *blockchain.OrganizationMetaData,
	srvMetaData *blockchain.ServiceMetadata, service FreeCallUserService, validator *FreeCallPaymentValidator) *FreeCallStateService {
	return &FreeCallStateService{orgMetadata: orgMetadata, serviceMetadata: srvMetaData, freeCallService: service, freeCallValidator: validator}
}

type BlockChainDisabledFreeCallStateService struct {
}

func (service *BlockChainDisabledFreeCallStateService) mustEmbedUnimplementedFreeCallStateServiceServer() {
	//TODO implement me
	panic("implement me")
}

func (service *FreeCallStateService) GetFreeCallsAvailable(context context.Context,
	request *FreeCallStateRequest) (reply *FreeCallStateReply, err error) {
	if err = service.verify(request); err != nil {
		zap.L().Error("Error in authorizing the request", zap.Error(err))
		return nil, err
	}
	availableCalls, err := service.checkForFreeCalls(service.getFreeCallPayment(request))
	if err != nil {
		return &FreeCallStateReply{}, err
	}
	return &FreeCallStateReply{UserId: request.UserId, FreeCallsAvailable: uint64(availableCalls)}, nil
}

func (service *BlockChainDisabledFreeCallStateService) GetFreeCallsAvailable(context.Context, *FreeCallStateRequest) (*FreeCallStateReply, error) {
	return &FreeCallStateReply{UserId: "", FreeCallsAvailable: 0}, fmt.Errorf("error in determining free calls because blockchain is disabled, contact service prodiver")
}

func (service *FreeCallStateService) verify(request *FreeCallStateRequest) (err error) {

	if err := service.freeCallValidator.Validate(service.getFreeCallPayment(request)); err != nil {
		return err
	}
	return nil
}

func (service *FreeCallStateService) checkForFreeCalls(payment *FreeCallPayment) (callsAvailable int, err error) {
	//Now get the state from etcd for this user, if there are no records, then return the free calls
	key, err := service.freeCallService.GetFreeCallUserKey(payment)
	if err != nil {
		return 0, err
	}
	data, ok, err := service.freeCallService.FreeCallUser(key)
	if !ok {
		return 0, fmt.Errorf("error in retrieving free call details from storage")
	}
	if err != nil {
		return 0, err
	}
	if freeCallsAllowed := config.GetFreeCallsCount(key.UserId); freeCallsAllowed > 0 {
		return freeCallsAllowed - data.FreeCallsMade, err
	}
	return service.serviceMetadata.GetFreeCallsAllowed() - data.FreeCallsMade, nil
}

func (service *FreeCallStateService) getFreeCallPayment(request *FreeCallStateRequest) *FreeCallPayment {
	return &FreeCallPayment{
		GroupId:                    service.orgMetadata.GetGroupIdString(),
		OrganizationId:             config.GetString(config.OrganizationId),
		ServiceId:                  config.GetString(config.ServiceId),
		UserId:                     request.GetUserId(),
		Signature:                  request.GetSignature(),
		CurrentBlockNumber:         big.NewInt(int64(request.GetCurrentBlock())),
		AuthToken:                  request.GetTokenForFreeCall(),
		AuthTokenExpiryBlockNumber: big.NewInt(int64(request.GetTokenExpiryDateBlock())),
	}
}
