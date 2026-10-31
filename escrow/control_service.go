//go:generate protoc -I . ./control_service.proto --go-grpc_out=paths=source_relative:. --go_out=paths=source_relative:.
package escrow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/utils"

	"go.uber.org/zap"
)

// ProviderControlService authenticates provider requests, lists authorizations,
// and prepares claims in local storage. Clients submit blockchain transactions.
type ProviderControlService struct {
	channelService       PaymentChannelService
	serviceMetaData      *blockchain.ServiceMetadata
	organizationMetaData *blockchain.OrganizationMetaData
	blockchain           blockchain.Processor
	mpeAddress           common.Address
}

func (service *ProviderControlService) mustEmbedUnimplementedProviderControlServiceServer() {
	// This marker satisfies the generated gRPC interface; RPC dispatch does not call it.
	panic("implement me")
}

// BlockChainDisabledProviderControlService returns empty replies without payment
// authentication or claim preparation when blockchain integration is disabled.
type BlockChainDisabledProviderControlService struct {
}

func (service *BlockChainDisabledProviderControlService) mustEmbedUnimplementedProviderControlServiceServer() {
	// This marker satisfies the generated gRPC interface; RPC dispatch does not call it.
	panic("implement me")
}

func (service *BlockChainDisabledProviderControlService) GetListUnclaimed(ctx context.Context, request *GetPaymentsListRequest) (paymentReply *PaymentsListReply, err error) {
	return &PaymentsListReply{}, nil
}

func (service *BlockChainDisabledProviderControlService) GetListInProgress(ctx context.Context, request *GetPaymentsListRequest) (reply *PaymentsListReply, err error) {
	return &PaymentsListReply{}, nil
}

func (service *BlockChainDisabledProviderControlService) StartClaim(ctx context.Context, startClaim *StartClaimRequest) (paymentReply *PaymentReply, err error) {
	return &PaymentReply{}, nil
}

func (service *BlockChainDisabledProviderControlService) StartClaimForMultipleChannels(ctx context.Context, request *StartMultipleClaimRequest) (reply *PaymentsListReply, err error) {
	return &PaymentsListReply{}, nil
}

func NewProviderControlService(blockchainProcessor blockchain.Processor, channelService PaymentChannelService, serMetaData *blockchain.ServiceMetadata,
	orgMetadata *blockchain.OrganizationMetaData) *ProviderControlService {
	return &ProviderControlService{
		channelService:       channelService,
		serviceMetaData:      serMetaData,
		organizationMetaData: orgMetadata,
		mpeAddress:           common.HexToAddress(serMetaData.MpeAddress),
		blockchain:           blockchainProcessor,
	}
}

// GetListUnclaimed authenticates the provider and lists nonzero authorizations
// from channel storage. The signed block must be within AllowedBlockDifference
// of the latest block, including both boundaries. Claim signatures are omitted;
// preparing a claim is a separate operation that advances local channel state.
func (service *ProviderControlService) GetListUnclaimed(ctx context.Context, request *GetPaymentsListRequest) (paymentReply *PaymentsListReply, err error) {
	// Require the contract configured in service metadata.
	if err := service.checkMpeAddress(request.GetMpeAddress()); err != nil {
		return nil, err
	}
	if err := service.blockchain.CompareWithLatestBlockNumber(big.NewInt(int64(request.CurrentBlock)), AllowedBlockDifference); err != nil {
		return nil, err
	}
	// Require the payment address configured in organization metadata.
	if err := service.verifySignerForListUnclaimed(request); err != nil {
		return nil, err
	}
	return service.listChannels()
}

// StartClaimForMultipleChannels authenticates one batch request, removes pending
// payments superseded on chain, and prepares claims in the original request order.
// The batch signature covers sorted channel IDs and a recent block; individual
// StartClaim request signatures are not required.
//
// Processing stops at the first error. Earlier channel updates remain stored,
// but the reply is populated only after all channels succeed. After an error,
// query GetListInProgress for claims already recorded. Clients remain responsible
// for submitting the returned payment signatures to the blockchain.
func (service *ProviderControlService) StartClaimForMultipleChannels(ctx context.Context, request *StartMultipleClaimRequest) (reply *PaymentsListReply, err error) {
	if err := service.checkMpeAddress(request.GetMpeAddress()); err != nil {
		return nil, err
	}
	if err := service.blockchain.CompareWithLatestBlockNumber(big.NewInt(int64(request.CurrentBlock)), AllowedBlockDifference); err != nil {
		return nil, err
	}
	if err := service.verifySignerForStartClaimForMultipleChannels(request); err != nil {
		return nil, err
	}
	err = service.removeClaimedPayments()
	if err != nil {
		zap.L().Error("unable to remove payments from which are already claimed")
		return nil, err
	}
	return service.startClaims(request)
}

func (service *ProviderControlService) startClaims(request *StartMultipleClaimRequest) (reply *PaymentsListReply, err error) {
	reply = &PaymentsListReply{}
	payments := make([]*PaymentReply, 0)

	for _, channelId := range request.GetChannelIds() {
		payment, err := service.beginClaimOnChannel(big.NewInt(int64(channelId)))
		if err != nil {
			// Earlier claims remain stored, but reply.Payments is assigned only
			// after the entire batch succeeds.
			return reply, err
		}
		payments = append(payments, payment)
	}
	reply.Payments = payments
	return reply, nil
}

// verifySignerForStartClaimForMultipleChannels checks the provider signature over
// "__StartClaimForMultipleChannels_" || MPE address || sorted channel IDs || block.
// The address is 20 bytes; each channel ID and the block use 32-byte encoding.
func (service *ProviderControlService) verifySignerForStartClaimForMultipleChannels(request *StartMultipleClaimRequest) error {
	message := bytes.Join([][]byte{
		[]byte("__StartClaimForMultipleChannels_"),
		service.serviceMetaData.GetMpeAddress().Bytes(),
		getBytesOfChannelIds(request),
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	return service.verifySigner(message, request.GetSignature())
}

func getBytesOfChannelIds(request *StartMultipleClaimRequest) []byte {
	channelIds := make([]uint64, 0)
	channelIds = append(channelIds, request.GetChannelIds()...)
	// Canonicalize only the signed payload. Claim execution retains request order.
	Uint64s(channelIds)
	channelIdInBytes := make([]byte, 0)
	for index, channelId := range channelIds {
		if index == 0 {
			channelIdInBytes = bytes.Join([][]byte{
				bigIntToBytes(big.NewInt(int64(channelId))),
			}, nil)
		} else {
			channelIdInBytes = bytes.Join([][]byte{
				channelIdInBytes,
				bigIntToBytes(big.NewInt(int64(channelId))),
			}, nil)
		}

	}
	return channelIdInBytes
}

type Uint64Slice []uint64

func Uint64s(l []uint64) {
	sort.Sort(Uint64Slice(l))
}

func (p Uint64Slice) Len() int           { return len(p) }
func (p Uint64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Uint64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// GetListInProgress authenticates the provider and recent block, removes pending
// records superseded by the blockchain nonce, and returns the remaining signed
// claim payments. Unlike GetListUnclaimed, these authorizations have already been
// separated from the current local channel state by claim preparation.
func (service *ProviderControlService) GetListInProgress(ctx context.Context, request *GetPaymentsListRequest) (reply *PaymentsListReply, err error) {
	if err := service.checkMpeAddress(request.GetMpeAddress()); err != nil {
		return nil, err
	}
	if err := service.blockchain.CompareWithLatestBlockNumber(big.NewInt(int64(request.CurrentBlock)), AllowedBlockDifference); err != nil {
		return nil, err
	}
	if err := service.verifySignerForListInProgress(request); err != nil {
		return nil, err
	}
	err = service.removeClaimedPayments()
	if err != nil {
		zap.L().Error("unable to remove payments from which are already claimed")
		return nil, err
	}
	return service.listClaims()
}

// StartClaim authenticates the provider against the channel's current nonce,
// removes pending records superseded on chain, and prepares one local claim.
// The reply contains the previous nonce's authorization; the local channel moves
// to the next nonce with the claimed amount deducted. This RPC does not submit a
// blockchain transaction, and its signed request uses a nonce rather than a block.
func (service *ProviderControlService) StartClaim(ctx context.Context, startClaim *StartClaimRequest) (paymentReply *PaymentReply, err error) {
	// Require the contract configured in service metadata.
	if err := service.checkMpeAddress(startClaim.MpeAddress); err != nil {
		return nil, err
	}
	// Require the payment address configured in organization metadata.
	err = service.verifySignerForStartClaim(startClaim)
	if err != nil {
		return nil, err
	}
	// Remove pending records whose nonce has been superseded on chain.
	err = service.removeClaimedPayments()
	if err != nil {
		zap.L().Error("unable to remove payments from etcd storage which are already claimed in block chain")
		return nil, err
	}
	return service.beginClaimOnChannel(bytesToBigInt(startClaim.GetChannelId()))
}

// listChannels lists stored authorizations without exposing their claim signatures.
func (service *ProviderControlService) listChannels() (*PaymentsListReply, error) {
	channels, err := service.channelService.ListChannels()
	if err != nil {
		return nil, err
	}
	output := make([]*PaymentReply, 0)
	for _, channel := range channels {
		// Skip channels with no authorized amount to claim.
		if channel.AuthorizedAmount.Int64() == 0 {
			continue
		}
		paymentReply := &PaymentReply{
			ChannelId:     bigIntToBytes(channel.ChannelID),
			ChannelNonce:  bigIntToBytes(channel.Nonce),
			SignedAmount:  bigIntToBytes(channel.AuthorizedAmount),
			ChannelExpiry: bigIntToBytes(channel.Expiration),
		}
		output = append(output, paymentReply)
	}
	paymentList := &PaymentsListReply{
		Payments: output,
	}
	return paymentList, nil
}

// verifySignerForListUnclaimed checks the provider signature over
// "__list_unclaimed" || MPE address || current block.
func (service *ProviderControlService) verifySignerForListUnclaimed(request *GetPaymentsListRequest) error {
	return service.verifySigner(service.getMessageBytes("__list_unclaimed", request), request.GetSignature())
}

func (service *ProviderControlService) getMessageBytes(prefixMessage string, request *GetPaymentsListRequest) []byte {
	message := bytes.Join([][]byte{
		[]byte(prefixMessage),
		service.serviceMetaData.GetMpeAddress().Bytes(),
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	return message
}

func (service *ProviderControlService) verifySigner(message []byte, signature []byte) error {
	signer, err := utils.GetSignerAddressFromMessage(message, signature)
	if err != nil {
		zap.L().Error(err.Error())
		return err
	}
	if err = utils.VerifyAddress(*signer, service.organizationMetaData.GetPaymentAddress()); err != nil {
		return err
	}
	return nil
}

// beginClaimOnChannel checks for an authorization and prepares its pending claim,
// advancing the stored channel nonce and resetting its current authorization.
func (service *ProviderControlService) beginClaimOnChannel(channelId *big.Int) (*PaymentReply, error) {
	latestChannel, ok, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: channelId})
	if err != nil {
		return nil, err
	}

	if !ok {
		err = fmt.Errorf("channel Id %v was not found on blockchain or storage", channelId)
		return nil, err
	}
	// Reject channels with no authorized amount to claim.
	if latestChannel.AuthorizedAmount.Int64() == 0 {
		err = fmt.Errorf("authorized amount is zero , hence nothing to claim on the channel Id: %v", channelId)
		return nil, err
	}
	claim, err := service.channelService.StartClaim(&PaymentChannelKey{ID: channelId}, IncrementChannelNonce)
	if err != nil {
		return nil, err
	}
	payment := claim.Payment()
	paymentReply := &PaymentReply{
		ChannelId:    bigIntToBytes(channelId),
		ChannelNonce: bigIntToBytes(payment.ChannelNonce),
		Signature:    payment.Signature,
		SignedAmount: bigIntToBytes(payment.Amount),
	}
	return paymentReply, nil
}

// verifySignerForStartClaim checks the provider signature over
// "__start_claim" || MPE address || channel ID || latest local channel nonce.
func (service *ProviderControlService) verifySignerForStartClaim(startClaim *StartClaimRequest) error {
	channelId := bytesToBigInt(startClaim.GetChannelId())
	signature := startClaim.Signature
	latestChannel, ok, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: channelId})
	if !ok || err != nil {
		return err
	}
	message := bytes.Join([][]byte{
		[]byte("__start_claim"),
		service.serviceMetaData.GetMpeAddress().Bytes(),
		bigIntToBytes(channelId),
		bigIntToBytes(latestChannel.Nonce),
	}, nil)
	return service.verifySigner(message, signature)
}

// listClaims returns nonzero signed pending payments whose channel state can be
// loaded. The payment retains its claim nonce; expiration comes from current state.
func (service *ProviderControlService) listClaims() (*PaymentsListReply, error) {
	claimsRetrieved, err := service.channelService.ListClaims()
	if err != nil {
		zap.L().Error("error in retrieving claims")
		return nil, err
	}
	output := make([]*PaymentReply, 0)
	for _, claimRetrieved := range claimsRetrieved {
		payment := claimRetrieved.Payment()
		// Read the latest available expiration without replacing the claim's nonce.
		latestChannel, ok, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: payment.ChannelID})
		if !ok || err != nil {
			zap.L().Error("Unable to retrieve the latest Channel State", zap.Any("ChanelID", payment.ChannelID), zap.Any("ChannelNonce", payment.ChannelNonce))
			continue
		}
		if payment.Signature == nil || payment.Amount.Int64() == 0 {
			zap.L().Error("The Signature or the Amount is not defined on the Payment with",
				zap.Any("ChannelID", payment.ChannelID),
				zap.Any("ChannelNonce", payment.ChannelNonce))
			continue
		}
		paymentReply := &PaymentReply{
			ChannelId:     bigIntToBytes(payment.ChannelID),
			ChannelNonce:  bigIntToBytes(payment.ChannelNonce),
			SignedAmount:  bigIntToBytes(payment.Amount),
			Signature:     payment.Signature,
			ChannelExpiry: bigIntToBytes(latestChannel.Expiration),
		}
		output = append(output, paymentReply)
	}
	reply := &PaymentsListReply{
		Payments: output,
	}
	return reply, nil
}

// verifySignerForListInProgress checks the provider signature over
// "__list_in_progress" || MPE address || current block.
func (service *ProviderControlService) verifySignerForListInProgress(request *GetPaymentsListRequest) error {
	return service.verifySigner(service.getMessageBytes("__list_in_progress", request), request.GetSignature())
}

// removeClaimedPayments deletes pending records whose payment nonce is lower than
// the current blockchain nonce. It compares against the stored payment's nonce,
// not the local channel nonce, which may already have advanced during preparation.
// No blockchain transaction is submitted by this cleanup.
func (service *ProviderControlService) removeClaimedPayments() error {
	claimsRetrieved, err := service.channelService.ListClaims()
	if err != nil {
		return errors.New("error in retrieving claims")
	}
	for _, claimRetrieved := range claimsRetrieved {
		payment := claimRetrieved.Payment()
		blockChainChannel, ok, err := service.channelService.PaymentChannelFromBlockChain(&PaymentChannelKey{ID: payment.ChannelID})
		if !ok || err != nil {
			return err
		}
		// The old authorization is no longer current once the chain nonce advances.
		if blockChainChannel.Nonce.Cmp(payment.ChannelNonce) > 0 {
			zap.L().Debug("the nonce of channel from Blockchain is greater than nonce of channel from etcd storage",
				zap.Any("ChannelID", payment.ChannelID),
				zap.Any("BlockchainNonce", blockChainChannel.Nonce),
				zap.Any("ChannelNonce", payment.ChannelNonce))
			err = claimRetrieved.Finish()
			if err != nil {
				zap.L().Error(err.Error())
				return err
			}
		}
	}
	return nil
}

// checkMpeAddress requires the contract address captured from service metadata.
func (service *ProviderControlService) checkMpeAddress(mpeAddress string) error {
	passedAddress := common.HexToAddress(mpeAddress)
	if !(service.mpeAddress == passedAddress) {
		return fmt.Errorf("the mpeAddress: %s passed does not match to what has been registered", mpeAddress)
	}
	return nil
}
