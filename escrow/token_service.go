//go:generate protoc -I . ./token_service.proto --go-grpc_out=paths=source_relative:. --go_out=paths=source_relative:.
package escrow

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/token"
	"github.com/singnet/snet-daemon/v6/utils"
)

// TokenService validates prepaid authorizations and issues channel-bound JWTs.
// Claim signatures authorize funds; the separate request signature adds freshness.
type TokenService struct {
	channelService      PaymentChannelService
	prePaidUsageService PrePaidService
	tokenManager        token.Manager
	validator           *ChannelPaymentValidator
	blockValidator      *blockchain.EthereumBlockValidator
	serviceMetaData     blockchain.ServiceMetadata
}

func (service *TokenService) mustEmbedUnimplementedTokenServiceServer() {
	// This marker satisfies the generated gRPC interface; RPC dispatch does not call it.
	panic("implement me")
}

// BlockChainDisabledTokenService returns an empty reply without issuing a JWT or
// validating payment authorization when blockchain integration is disabled.
type BlockChainDisabledTokenService struct {
}

func (service BlockChainDisabledTokenService) mustEmbedUnimplementedTokenServiceServer() {
	// This marker satisfies the generated gRPC interface; RPC dispatch does not call it.
	panic("implement me")
}

func (service BlockChainDisabledTokenService) GetToken(ctx context.Context, request *TokenRequest) (reply *TokenReply, err error) {
	return &TokenReply{}, nil
}

// NewTokenService uses blockValidator for token-request freshness independently
// of the channel-payment validator's block source and expiration check.
func NewTokenService(paymentChannelService PaymentChannelService,
	usageService PrePaidService, tokenManager token.Manager, validator *ChannelPaymentValidator,
	blockValidator *blockchain.EthereumBlockValidator, metadata *blockchain.ServiceMetadata) *TokenService {

	return &TokenService{
		channelService:      paymentChannelService,
		prePaidUsageService: usageService,
		tokenManager:        tokenManager,
		validator:           validator,
		blockValidator:      blockValidator,
		serviceMetaData:     *metadata,
	}
}

func (service *TokenService) verifySignatureAndSignedAmountEligibility(channelId *big.Int,
	latestAuthorizedAmount *big.Int, request *TokenRequest) (singer *common.Address, err error) {
	channel, ok, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: channelId})

	if !ok {
		return nil, fmt.Errorf("channel is not found, channelId: %v", channelId)
	}
	if err != nil {
		return nil, fmt.Errorf("error:%v was seen on retrieving details of channelID:%v",
			err.Error(), channelId)
	}
	if channel.AuthorizedAmount.Cmp(latestAuthorizedAmount) > 0 {
		return nil, fmt.Errorf("signed amount for token request needs to be greater than last signed amount")
	}
	if channel.FullAmount.Cmp(latestAuthorizedAmount) < 0 {
		return nil, fmt.Errorf("signed amount for token request cannot be greater than full amount in channel")
	}
	// Authenticate the fresh token request separately from its claim authorization.
	signer, err := service.verifySignature(request, channel)
	if err != nil {
		return nil, err
	}
	payment := service.getPayment(channelId, latestAuthorizedAmount, request)
	if err = service.validator.Validate(payment, channel); err != nil {
		return nil, err
	}
	// A higher cumulative authorization is committed to channel storage, then
	// its increase is added to the prepaid planned counter. These are separate
	// writes; an unchanged authorization skips both updates.
	if latestAuthorizedAmount.Cmp(channel.AuthorizedAmount) > 0 {
		transaction, err := service.channelService.StartPaymentTransaction(payment)
		if err != nil {
			return nil, err
		}
		if err = transaction.Commit(); err != nil {
			return nil, err
		}
		if err = service.prePaidUsageService.UpdateUsage(channelId, latestAuthorizedAmount.Sub(latestAuthorizedAmount, channel.AuthorizedAmount), PLANNED_AMOUNT); err != nil {
			return nil, err
		}
	}

	return signer, nil
}

func (service *TokenService) getPayment(channelId *big.Int, latestAuthorizedAmount *big.Int, request *TokenRequest) *Payment {
	return &Payment{
		MpeContractAddress: service.serviceMetaData.GetMpeAddress(),
		ChannelID:          channelId,
		ChannelNonce:       big.NewInt(0).SetUint64(request.CurrentNonce),
		Amount:             latestAuthorizedAmount,
		Signature:          request.ClaimSignature,
	}
}

// verifySignature authenticates claim-signature bytes || current block, allowing
// the channel signer, sender, or recipient to request a token. The embedded claim
// signature is checked separately by ChannelPaymentValidator.
func (service *TokenService) verifySignature(request *TokenRequest, channel *PaymentChannelData) (send *common.Address, err error) {
	message := bytes.Join([][]byte{
		request.GetClaimSignature(),
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	signature := request.GetSignature()

	sender, err := utils.GetSignerAddressFromMessage(message, signature)
	if err != nil {
		return nil, fmt.Errorf("incorrect signature")
	}

	if channel.Signer != *sender && *sender != channel.Sender && *sender != channel.Recipient {
		return nil, fmt.Errorf("only channel signer/sender/receiver can get a Valid Token")
	}

	if err = service.blockValidator.ValidateBlock(new(big.Int).SetUint64(request.CurrentBlock), AllowedBlockDifference); err != nil {
		return nil, err
	}
	return sender, nil
}

// GetToken validates authorization, updates prepaid credit if the signed amount
// increases, and returns a JWT with stored planned and used totals. Renewing with
// an unchanged amount requires an existing planned counter; spending limits are
// enforced when prepaid calls are charged, not when the token is issued.
func (service *TokenService) GetToken(ctx context.Context, request *TokenRequest) (reply *TokenReply, err error) {

	channelID := big.NewInt(0).SetUint64(request.ChannelId)
	latestAuthorizedAmount := big.NewInt(0).SetUint64(request.SignedAmount)

	signer, err := service.verifySignatureAndSignedAmountEligibility(channelID, latestAuthorizedAmount, request)
	if err != nil {
		return nil, err
	}

	usage, ok, err := service.prePaidUsageService.GetUsage(PrePaidDataKey{ChannelID: channelID, UsageType: USED_AMOUNT})
	usageAmount := big.NewInt(0)
	if ok {
		usageAmount = usage.Amount
	}
	if err != nil {
		return nil, err
	}

	plannedAmount, ok, err := service.prePaidUsageService.GetUsage(PrePaidDataKey{ChannelID: channelID, UsageType: PLANNED_AMOUNT})
	if !ok {
		return nil, fmt.Errorf("unable to retrieve planned Amount %v", err)
	}
	if err != nil {
		return nil, err
	}
	tokenGenerated, err := service.tokenManager.CreateToken(channelID, signer.Hex())
	return &TokenReply{ChannelId: request.ChannelId, Token: fmt.Sprintf("%v", tokenGenerated), PlannedAmount: plannedAmount.Amount.Uint64(),
		UsedAmount: usageAmount.Uint64()}, err
}
