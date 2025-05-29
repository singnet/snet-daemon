//go:generate protoc -I . ./token_service.proto --go-grpc_out=. --go_out=.
package escrow

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/singnet/snet-daemon/v5/authutils"
	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/token"
	"golang.org/x/net/context"
	"math/big"
)

type TokenService struct {
	channelService          PaymentChannelService
	prePaidUsageService     PrePaidService
	tokenManager            token.Manager
	validator               *ChannelPaymentValidator
	serviceMetaData         blockchain.ServiceMetadata
	allowedBlockNumberCheck func(blockNumber *big.Int) (err error)
}

func (service *TokenService) mustEmbedUnimplementedTokenServiceServer() {
	//TODO implement me
	panic("implement me")
}

type BlockChainDisabledTokenService struct {
}

func (service BlockChainDisabledTokenService) mustEmbedUnimplementedTokenServiceServer() {
	//TODO implement me
	panic("implement me")
}

func (service BlockChainDisabledTokenService) GetToken(ctx context.Context, request *TokenRequest) (reply *TokenReply, err error) {
	return &TokenReply{}, nil
}

func NewTokenService(paymentChannelService PaymentChannelService,
	usageService PrePaidService, tokenManager token.Manager, validator *ChannelPaymentValidator, metadata *blockchain.ServiceMetadata) *TokenService {

	return &TokenService{
		channelService:      paymentChannelService,
		prePaidUsageService: usageService,
		tokenManager:        tokenManager,
		validator:           validator,
		serviceMetaData:     *metadata,
		allowedBlockNumberCheck: func(blockNumber *big.Int) error {
			currentBlockNumber, err := validator.currentBlock()
			if err != nil {
				return err
			}
			differenceInBlockNumber := blockNumber.Sub(blockNumber, currentBlockNumber)
			if differenceInBlockNumber.Abs(differenceInBlockNumber).Uint64() > authutils.AllowedBlockChainDifference {
				return fmt.Errorf("authentication failed as the signature passed has expired")
			}
			return nil
		},
	}
}

func (service *TokenService) verifySignatureAndSignedAmountEligibility(channelId *big.Int,
	latestAuthorizedAmount *big.Int, request *TokenRequest) (err error) {
	channel, ok, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: channelId})

	if !ok {
		return fmt.Errorf("channel is not found, channelId: %v", channelId)
	}
	if err != nil {
		return fmt.Errorf("error:%v was seen on retreiving details of channelID:%v",
			err.Error(), channelId)
	}
	if channel.AuthorizedAmount.Cmp(latestAuthorizedAmount) > 0 {
		return fmt.Errorf("signed amount for token request needs to be greater than last signed amount")
	}
	if channel.FullAmount.Cmp(latestAuthorizedAmount) < 0 {
		return fmt.Errorf("signed amount for token request cannot be greater than full amount in channel")
	}
	//verify signature
	if err = service.verifySignature(request, channel); err != nil {
		return err
	}
	payment := service.getPayment(channelId, latestAuthorizedAmount, request)
	if err = service.validator.Validate(payment, channel); err != nil {
		return err
	}
	//update the channel Signature if you have a new Signed Amount received
	if latestAuthorizedAmount.Cmp(channel.AuthorizedAmount) > 0 {
		transaction, err := service.channelService.StartPaymentTransaction(payment)
		if err != nil {
			return err
		}
		if err = transaction.Commit(); err != nil {
			return err
		}
		if err = service.prePaidUsageService.UpdateUsage(channelId, latestAuthorizedAmount.Sub(latestAuthorizedAmount, channel.AuthorizedAmount), PLANNED_AMOUNT); err != nil {
			return err
		}
	}

	return nil
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

func (service *TokenService) verifySignature(request *TokenRequest, channel *PaymentChannelData) (err error) {
	message := bytes.Join([][]byte{
		request.GetClaimSignature(),
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	signature := request.GetSignature()

	sender, err := authutils.GetSignerAddressFromMessage(message, signature)
	if err != nil {
		return fmt.Errorf("incorrect signature")
	}

	if channel.Signer != *sender && *sender != channel.Sender && *sender != channel.Recipient {
		return fmt.Errorf("only channel signer/sender/receiver can get a Valid Token")
	}

	if err = service.allowedBlockNumberCheck(big.NewInt(0).SetUint64(request.CurrentBlock)); err != nil {
		return err
	}
	return nil
}

func (service *TokenService) GetToken(ctx context.Context, request *TokenRequest) (reply *TokenReply, err error) {

	//Check for update state
	channelID := big.NewInt(0).SetUint64(request.ChannelId)
	latestAuthorizedAmount := big.NewInt(0).SetUint64(request.SignedAmount)

	if err = service.verifySignatureAndSignedAmountEligibility(channelID, latestAuthorizedAmount, request); err != nil {
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
	tokenGenerated, err := service.tokenManager.CreateToken(channelID)
	return &TokenReply{ChannelId: request.ChannelId, Token: fmt.Sprintf("%v", tokenGenerated), PlannedAmount: plannedAmount.Amount.Uint64(),
		UsedAmount: usageAmount.Uint64()}, err
}
