//go:generate protoc -I . ./control_service.proto --go-grpc_out=. --go_out=.
package escrow

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/utils"

	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type ProviderControlService struct {
	channelService       PaymentChannelService
	serviceMetaData      *blockchain.ServiceMetadata
	organizationMetaData *blockchain.OrganizationMetaData
	blockchain           blockchain.Processor
	mpeAddress           common.Address
}

func (service *ProviderControlService) mustEmbedUnimplementedProviderControlServiceServer() {
	//TODO implement me
	panic("implement me")
}

type BlockChainDisabledProviderControlService struct {
}

func (service *BlockChainDisabledProviderControlService) mustEmbedUnimplementedProviderControlServiceServer() {
	//TODO implement me
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

/*
	GetListUnclaimed

Get list of unclaimed payments, we do this by getting the list of channels in progress which have some amount to be claimed.
Verify that mpe_address is correct
Verify that actual block_number is not very different (+-5 blocks) from the current_block_number from the signature
Verify that message was signed by the service provider (“payment_address” in metadata should match to the signer).
Send list of unclaimed payments
*/
func (service *ProviderControlService) GetListUnclaimed(ctx context.Context, request *GetPaymentsListRequest) (paymentReply *PaymentsListReply, err error) {
	//Check if the mpe address matches to what is there in service metadata
	if err := service.checkMpeAddress(request.GetMpeAddress()); err != nil {
		return nil, err
	}
	if err := service.blockchain.CompareWithLatestBlockNumber(big.NewInt(int64(request.CurrentBlock)), AllowedBlockDifference); err != nil {
		return nil, err
	}
	//Check if the signer is valid
	if err := service.verifySignerForListUnclaimed(request); err != nil {
		return nil, err
	}
	return service.listChannels()
}

// StartClaimForMultipleChannels starts claim processes for multiple payment channels.
//
// Typical flow: a client first queries the daemon for all pending claims and then
// initiates StartClaim for each selected channel. Each StartClaim requires a fresh
// signature.
//
// This method:
//  1. Validates the provided MPE address.
//  2. Ensures the provided block number is within AllowedBlockDifference of the latest block.
//  3. Verifies the signer for the multi-channel request.
//  4. Removes payments that have already been claimed.
//  5. Iterates over the channel IDs in the request and internally invokes StartClaim
//     for each channel, in order.
//
// On error, processing stops at the first failing channel. The returned
// PaymentsListReply contains all successfully started claims up to that point,
// and the encountered error is returned alongside.
//
// Note: the underlying blockchain can settle multiple claims in a single transaction
// (multiChannelClaim). This API only initiates the per-channel “start” phase; clients
// may still aggregate on-chain settlement as needed.
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
			// we stop here and return back what ever was successful as reply
			return reply, err
		}
		payments = append(payments, payment)
	}
	reply.Payments = payments
	return reply, nil
}

// ("__StartClaimForMultipleChannels_, mpe_address,channel_id1,channel_id2,...,current_block_number)
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
	//sort the channel Ids
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

// Get the list of all claims that have been initiated but not completed yet.
// Verify that mpe_address is correct
// Verify that actual block_number is not very different (+-5 blocks) from the current_block_number from the signature
// Verify that message was signed by the service provider (“payment_address” in metadata should match to the signer).
// Check for any claims already done on block chain but have not been reflected in the storage yet,
// update the storage status by calling the Finish() method on such claims.
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

// Initialize the claim for specific channel
// Verify that the “payment_address” in meta data matches to that of the signer.
// Increase nonce and send last payment with old nonce to the caller.
// Begin the claim process on the current channel and Increment the channel nonce and
// decrease the full amount to allow channel sender to continue working with remaining amount.
// Check for any claims already done on block chain but have not been reflected in the storage yet,
// update the storage status by calling the Finish() method on such claims
func (service *ProviderControlService) StartClaim(ctx context.Context, startClaim *StartClaimRequest) (paymentReply *PaymentReply, err error) {
	// Check if the mpe address matches to what is there in service metadata
	if err := service.checkMpeAddress(startClaim.MpeAddress); err != nil {
		return nil, err
	}
	// Verify signature, check if “payment_address” matches to what is there in metadata
	err = service.verifySignerForStartClaim(startClaim)
	if err != nil {
		return nil, err
	}
	// Remove any payments already claimed on blockchain
	err = service.removeClaimedPayments()
	if err != nil {
		zap.L().Error("unable to remove payments from etcd storage which are already claimed in block chain")
		return nil, err
	}
	return service.beginClaimOnChannel(bytesToBigInt(startClaim.GetChannelId()))
}

// get the list of channels in progress which have some amount to be claimed.
func (service *ProviderControlService) listChannels() (*PaymentsListReply, error) {
	// get the list of channels from storage in progress which have some amount to be claimed.
	channels, err := service.channelService.ListChannels()
	if err != nil {
		return nil, err
	}
	output := make([]*PaymentReply, 0)
	for _, channel := range channels {
		//ignore if nothing is to be claimed
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

// message used to sign is of the form ("__list_unclaimed", mpe_address, current_block_number)
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

// Begin the claim process on the current channel and Increment the channel nonce and
// decrease the full amount to allow channel sender to continue working with the remaining amount.
func (service *ProviderControlService) beginClaimOnChannel(channelId *big.Int) (*PaymentReply, error) {
	latestChannel, ok, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: channelId})
	if err != nil {
		return nil, err
	}

	if !ok {
		err = fmt.Errorf("channel Id %v was not found on blockchain or storage", channelId)
		return nil, err
	}
	//Check if there is any Authorized amount to initiate a claim
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

// Verify if the signer is the same as the payment address in metadata
// __start_claim”, mpe_address, channel_id, channel_nonce
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

func (service *ProviderControlService) listClaims() (*PaymentsListReply, error) {
	//retrieve all the claims in progress
	claimsRetrieved, err := service.channelService.ListClaims()
	if err != nil {
		zap.L().Error("error in retrieving claims")
		return nil, err
	}
	output := make([]*PaymentReply, 0)
	for _, claimRetrieved := range claimsRetrieved {
		payment := claimRetrieved.Payment()
		// To Get the Expiration of the Channel (always get the latest state)
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

// message used to sign is of the form ("__list_in_progress", mpe_address, current_block_number)
func (service *ProviderControlService) verifySignerForListInProgress(request *GetPaymentsListRequest) error {
	return service.verifySigner(service.getMessageBytes("__list_in_progress", request), request.GetSignature())
}

// No write operation on blockchain are done by Daemon (will be take care of by the snet client)
// Finish on the claim should be called only after the payment is successfully claimed and blockchain is updated accordingly.
// One way to determine this is by checking the nonce in the blockchain with the nonce in the payment,
// for a given channel if the blockchain nonce is greater than that of the nonce from etcd storage => that the claim is already done in blockchain.
// and the Finish method is called on the claim.
func (service *ProviderControlService) removeClaimedPayments() error {
	//Get the pending claims
	//retrieve all the claims in progress
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
		//Compare the state of this payment in progress with what is available in blockchain
		if blockChainChannel.Nonce.Cmp(payment.ChannelNonce) > 0 {
			//if the Nonce on this block chain is higher than that of the Payment,
			//means that the payment has been completed , hence update the etcd state with this
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

// Check if the mpe address passed matches to what is present in the metadata.
func (service *ProviderControlService) checkMpeAddress(mpeAddress string) error {
	passedAddress := common.HexToAddress(mpeAddress)
	if !(service.mpeAddress == passedAddress) {
		return fmt.Errorf("the mpeAddress: %s passed does not match to what has been registered", mpeAddress)
	}
	return nil
}
