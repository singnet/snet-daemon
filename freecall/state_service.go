// go generate runs in this package's directory. Use the module root as the
// include path so the descriptor is named "freecall/state_service.proto";
// "-I ." would register "state_service.proto", colliding with escrow's descriptor.
//
//go:generate protoc -I .. freecall/state_service.proto --go-grpc_out=paths=source_relative:.. --go_out=paths=source_relative:..

package freecall

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

// StateService issues provider-signed free-call tokens and reports local usage
// allowances. It reads the configured ERC-20 token for wallet eligibility, not
// MPE balances or channels, and does not submit blockchain transactions.
type StateService struct {
	UnimplementedFreeCallStateServiceServer
	orgMetadata           *blockchain.OrganizationMetaData
	serviceMetadata       *blockchain.ServiceMetadata
	freeCallService       UserService
	freeCallValidator     *PaymentValidator
	tokenInstance         ERC20
	minBalanceForFreeCall *big.Int // Whole token units, converted with an assumed 18 decimals.
}

// ERC20 exposes the wallet-balance read needed for free-call token eligibility.
// It is the network token contract, not an MPE contract or an escrow account.
type ERC20 interface {
	BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error)
}

// CheckBalanceForFreeCall requires the caller's wallet token balance to meet
// the configured whole-token minimum. It does not read MPE deposits, channel
// balances, ERC-20 allowances, or a native-currency balance.
func (service *StateService) CheckBalanceForFreeCall(ctx context.Context, address common.Address) error {

	balance, err := service.tokenInstance.BalanceOf(&bind.CallOpts{Context: ctx}, address)
	if err != nil {
		zap.L().Error("error can't get balance", zap.Error(err))
		return handler.NewGrpcErrorf(codes.PermissionDenied, "you must have at least %s FET (ASI) in your balance to use free calls", service.minBalanceForFreeCall.String())
	}

	// Convert the configured whole-token threshold to ERC-20 base units. The
	// implementation assumes 18 decimals rather than querying the token contract.
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18)), nil)
	threshold := new(big.Int).Mul(service.minBalanceForFreeCall, factor)

	if balance.Cmp(threshold) < 0 {
		return handler.NewGrpcErrorf(codes.PermissionDenied, "you must have at least %s FET (ASI) in your balance to use free calls", service.minBalanceForFreeCall.String())
	}
	return nil
}

// GetFreeCallToken authenticates a recent caller request and issues a token.
// Only configured trusted backends may supply a nonempty user ID; they also skip
// the wallet-balance check. Issuance does not check or reserve remaining calls,
// reset usage, or create an MPE payment authorization.
func (service *StateService) GetFreeCallToken(ctx context.Context, request *GetFreeCallTokenRequest) (*FreeCallToken, error) {

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

	// Trust is checked against the backend list itself. Being the provider's
	// free-call token signer alone does not grant this backend exemption.
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

	token, block := service.freeCallValidator.NewToken(request.Address, request.UserId, request.TokenLifetimeInBlocks)
	return &FreeCallToken{
		TokenHex:             hex.EncodeToString(token),
		Token:                token,
		TokenExpirationBlock: block.Uint64(),
	}, nil
}

// NewStateService uses the selected organization's group ID for caller signatures
// and the service group's default quota for availability. minBalanceForFreeCall
// is expressed in whole tokens. Daemon components supply the network ERC-20
// client and a signing key matching the service metadata's free-call signer.
func NewStateService(orgMetadata *blockchain.OrganizationMetaData,
	srvMetaData *blockchain.ServiceMetadata,
	service UserService,
	validator *PaymentValidator,
	tokenInstance ERC20, minBalanceForFreeCall *big.Int) *StateService {
	return &StateService{
		orgMetadata:           orgMetadata,
		serviceMetadata:       srvMetaData,
		freeCallService:       service,
		freeCallValidator:     validator,
		minBalanceForFreeCall: minBalanceForFreeCall,
		tokenInstance:         tokenInstance}
}

// GetFreeCallsAvailable validates the caller and token, then reads the remaining
// allowance. It neither locks nor consumes calls; token renewal does not reset
// the stored usage used for this calculation.
func (service *StateService) GetFreeCallsAvailable(context context.Context,
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

func (service *StateService) verify(payment *Payment) (err error) {
	if err := service.freeCallValidator.Validate(payment); err != nil {
		return err
	}
	return nil
}

// checkForFreeCalls subtracts committed usage from the per-address override or
// metadata quota, clamping exhausted quotas to zero. A configured unlimited
// override is represented by the legacy sentinel 99999999, not a finite quota.
func (service *StateService) checkForFreeCalls(payment *Payment) (callsAvailable uint64, err error) {
	// The supplied storage may be etcd or memory. User returns an unpersisted
	// zero-count entry for an identity with no record yet.
	key, err := service.freeCallService.UserKey(payment)
	if err != nil {
		return 0, err
	}
	data, ok, err := service.freeCallService.User(key)
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

func (service *StateService) getFreeCallPayment(request *FreeCallStateRequest) (*Payment, error) {
	parsedToken, block, err := ParseToken(request.FreeCallToken)
	if err != nil {
		return nil, err
	}
	return &Payment{
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

// DisabledStateService rejects token and availability RPCs when blockchain
// integration or the required free-call signing setup is unavailable.
type DisabledStateService struct {
	UnimplementedFreeCallStateServiceServer
}

func (service *DisabledStateService) GetFreeCallToken(ctx context.Context, request *GetFreeCallTokenRequest) (*FreeCallToken, error) {
	return &FreeCallToken{}, fmt.Errorf("error in generating token because blockchain is disabled, contact service provider")
}

func (service *DisabledStateService) GetFreeCallsAvailable(context.Context, *FreeCallStateRequest) (*FreeCallStateReply, error) {
	return &FreeCallStateReply{FreeCallsAvailable: 0}, fmt.Errorf("error in determining free calls because blockchain is disabled, contact service provider")
}
