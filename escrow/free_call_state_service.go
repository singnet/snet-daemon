package escrow

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type FreeCallStateService struct {
	orgMetadata           *blockchain.OrganizationMetaData
	serviceMetadata       *blockchain.ServiceMetadata
	freeCallService       FreeCallUserService
	freeCallValidator     *FreeCallPaymentValidator
	tokenInstance         ERC20
	minBalanceForFreeCall *big.Int // in asi, not aasi
}

type ERC20 interface {
	BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error)
}

func (service *FreeCallStateService) CheckBalanceForFreeCall(ctx context.Context, address common.Address) error {

	balance, err := service.tokenInstance.BalanceOf(&bind.CallOpts{Context: ctx}, address)
	if err != nil {
		zap.L().Error("error can't get balance", zap.Error(err))
		return handler.NewGrpcErrorf(codes.PermissionDenied, "you must have at least %s FET (ASI) in your balance to use free calls", service.minBalanceForFreeCall.String())
	}

	// 10 * 10^18
	// convert to aasi
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18)), nil)
	threshold := new(big.Int).Mul(service.minBalanceForFreeCall, factor)

	if balance.Cmp(threshold) < 0 {
		return handler.NewGrpcErrorf(codes.PermissionDenied, "you must have at least %s FET (ASI) in your balance to use free calls", service.minBalanceForFreeCall.String())
	}
	return nil
}

func (service *FreeCallStateService) GetFreeCallToken(ctx context.Context, request *GetFreeCallTokenRequest) (*FreeCallToken, error) {

	signer, err := getAddressFromSignatureForNewFreeCallToken(request, service.orgMetadata.GetGroupIdString())
	if err != nil {
		return nil, err
	}

	if *signer != common.HexToAddress(request.GetAddress()) {
		return nil, fmt.Errorf("invalid signer, %v (from request) is not equal to %v (signer)", request.GetAddress(), signer)
	}

	err = service.freeCallValidator.compareWithLatestBlockNumber(big.NewInt(0).SetUint64(request.GetCurrentBlock()))
	if err != nil {
		return nil, err
	}

	// If address is not trusted we can't allow user-id in request
	if !slices.ContainsFunc(service.freeCallValidator.trustedFreeCallSignerAddresses,
		func(addr common.Address) bool {
			return *signer == addr
		}) {
		if request.GetUserId() != "" {
			return nil, fmt.Errorf("your address is not trusted by this service provider, the use of user_id is not allowed")
		}
		err := service.CheckBalanceForFreeCall(ctx, common.HexToAddress(request.Address))
		if err != nil {
			return nil, err
		}
	}

	token, block := service.freeCallValidator.NewFreeCallToken(request.Address, request.UserId, request.TokenLifetimeInBlocks)
	return &FreeCallToken{
		TokenHex:             hex.EncodeToString(token), // string
		Token:                token,                     // bytes
		TokenExpirationBlock: block.Uint64(),
	}, nil
}

func (service *FreeCallStateService) mustEmbedUnimplementedFreeCallStateServiceServer() {
	//TODO implement me
	panic("implement me")
}

func NewFreeCallStateService(orgMetadata *blockchain.OrganizationMetaData,
	srvMetaData *blockchain.ServiceMetadata,
	service FreeCallUserService,
	validator *FreeCallPaymentValidator,
	tokenInstance ERC20, minBalanceForFreeCall *big.Int) *FreeCallStateService {
	return &FreeCallStateService{
		orgMetadata:           orgMetadata,
		serviceMetadata:       srvMetaData,
		freeCallService:       service,
		freeCallValidator:     validator,
		minBalanceForFreeCall: minBalanceForFreeCall,
		tokenInstance:         tokenInstance}
}

func (service *FreeCallStateService) GetFreeCallsAvailable(context context.Context,
	request *FreeCallStateRequest) (reply *FreeCallStateReply, err error) {

	payment, err := service.getFreeCallPayment(request)
	if err != nil {
		return nil, err
	}

	if err = service.verify(payment); err != nil {
		zap.L().Error("Error in authorizing the request", zap.Error(err))
		return nil, err
	}

	availableCalls, err := service.checkForFreeCalls(payment)
	if err != nil {
		return &FreeCallStateReply{}, err
	}
	return &FreeCallStateReply{FreeCallsAvailable: availableCalls}, nil
}

func (service *FreeCallStateService) verify(payment *FreeCallPayment) (err error) {
	if err := service.freeCallValidator.Validate(payment); err != nil {
		return err
	}
	return nil
}

func (service *FreeCallStateService) checkForFreeCalls(payment *FreeCallPayment) (callsAvailable uint64, err error) {
	//Now get the state from etcd for this user, if there are no records, then return the free calls
	key, err := service.freeCallService.GetFreeCallUserKey(payment)
	if err != nil {
		return 0, err
	}
	data, ok, err := service.freeCallService.FreeCallUser(key)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("error in retrieving free call details from storage")
	}

	freeCallsAllowed := config.GetFreeCallsAllowed(key.Address)
	if freeCallsAllowed == -1 {
		return 99999999, nil
	}

	if freeCallsAllowed > 0 {
		if freeCallsAllowed-data.FreeCallsMade < 0 {
			return 0, nil
		}
		return uint64(freeCallsAllowed - data.FreeCallsMade), err
	}

	freeCallsAllowed = service.serviceMetadata.GetFreeCallsAllowed() - data.FreeCallsMade
	if freeCallsAllowed < 0 {
		return 0, nil
	}

	return uint64(freeCallsAllowed), nil
}

func (service *FreeCallStateService) getFreeCallPayment(request *FreeCallStateRequest) (*FreeCallPayment, error) {
	parsedToken, block, err := ParseFreeCallToken(request.FreeCallToken)
	if err != nil {
		return nil, err
	}
	return &FreeCallPayment{
		GroupId:                    service.orgMetadata.GetGroupIdString(),
		OrganizationId:             config.GetString(config.OrganizationId),
		ServiceId:                  config.GetString(config.ServiceId),
		Address:                    request.GetAddress(),
		UserID:                     request.GetUserId(),
		Signature:                  request.GetSignature(),
		CurrentBlockNumber:         big.NewInt(int64(request.GetCurrentBlock())),
		AuthToken:                  request.GetFreeCallToken(),
		AuthTokenParsed:            parsedToken,
		AuthTokenExpiryBlockNumber: block,
	}, nil
}

type BlockChainDisabledFreeCallStateService struct {
}

func (service *BlockChainDisabledFreeCallStateService) GetFreeCallToken(ctx context.Context, request *GetFreeCallTokenRequest) (*FreeCallToken, error) {
	return &FreeCallToken{}, fmt.Errorf("error in generating token because blockchain is disabled, contact service provider")
}

func (service *BlockChainDisabledFreeCallStateService) mustEmbedUnimplementedFreeCallStateServiceServer() {
	//TODO implement me
	panic("implement me")
}

func (service *BlockChainDisabledFreeCallStateService) GetFreeCallsAvailable(context.Context, *FreeCallStateRequest) (*FreeCallStateReply, error) {
	return &FreeCallStateReply{FreeCallsAvailable: 0}, fmt.Errorf("error in determining free calls because blockchain is disabled, contact service provider")
}
