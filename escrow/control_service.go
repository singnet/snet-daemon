//go:generate protoc -I . ./control_service.proto --go_out=plugins=grpc:.
package escrow

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/singnet/snet-daemon/authutils"
	"github.com/singnet/snet-daemon/blockchain"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"math/big"
	"strings"
)

type ProviderControlService struct {
	channelService  PaymentChannelService
	serviceMetaData *blockchain.ServiceMetadata
}

func NewProviderControlService(channelService PaymentChannelService, metaData *blockchain.ServiceMetadata) *ProviderControlService {
	return &ProviderControlService{
		channelService:  channelService,
		serviceMetaData: metaData,
	}
}

/*
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
	if err := authutils.CompareWithLatestBlockNumber(big.NewInt(int64(request.CurrentBlock))); err != nil {
		return nil, err
	}
	//Check if the signer is valid
	if err := service.verifySignerForListUnclaimed(request); err != nil {
		return nil, err
	}
	return service.listChannels()
}

//Get the list of all claims that have been initiated but not completed yet.
//Verify that mpe_address is correct
//Verify that actual block_number is not very different (+-5 blocks) from the current_block_number from the signature
//Verify that message was signed by the service provider (“payment_address” in metadata should match to the signer).
//Check for any claims already done on block chain but have not been reflected in the storage yet,
//update the storage status by calling the Finish() method on such claims.
func (service *ProviderControlService) GetListInProgress(ctx context.Context, request *GetPaymentsListRequest) (reply *PaymentsListReply, err error) {

	if err := service.checkMpeAddress(request.GetMpeAddress()); err != nil {
		return nil, err
	}

	if err := authutils.CompareWithLatestBlockNumber(big.NewInt(int64(request.CurrentBlock))); err != nil {
		return nil, err
	}

	if err := service.verifySignerForListInProgress(request); err != nil {
		return nil, err
	}
	err = service.removeClaimedPayments()
	if err != nil {
		log.Errorf("unable to remove payments from which are already claimed")
		return nil, err
	}
	return service.listClaims()
}

//Initialize the claim for specific channel
//Verify that the “payment_address” in meta data matches to that of the signer.
//Increase nonce and send last payment with old nonce to the caller.
//Begin the claim process on the current channel and Increment the channel nonce and
//decrease the full amount to allow channel sender to continue working with remaining amount.
//Check for any claims already done on block chain but have not been reflected in the storage yet,
//update the storage status by calling the Finish() method on such claims
func (service *ProviderControlService) StartClaim(ctx context.Context, startClaim *StartClaimRequest) (paymentReply *PaymentReply, err error) {
	//Check if the mpe address matches to what is there in service metadata
	if err := service.checkMpeAddress(startClaim.MpeAddress); err != nil {
		return nil, err
	}
	//Verify signature , check if “payment_address” matches to what is there in metadata
	err = service.verifySignerForStartClaim(startClaim)
	if err != nil {
		return nil, err
	}
	//Remove any payments already claimed on block chain
	err = service.removeClaimedPayments()
	if err != nil {
		log.Error("unable to remove payments from etcd storage which are already claimed in block chain")
		return nil, err
	}
	return service.beginClaimOnChannel(bytesToBigInt(startClaim.GetChannelId()))
}

//get the list of channels in progress which have some amount to be claimed.
func (service *ProviderControlService) listChannels() (*PaymentsListReply, error) {
	//get the list of channels in progress which have some amount to be claimed.
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
			ChannelId:    bigIntToBytes(channel.ChannelID),
			ChannelNonce: bigIntToBytes(channel.Nonce),
			SignedAmount: bigIntToBytes(channel.AuthorizedAmount),
		}
		output = append(output, paymentReply)
	}
	paymentList := &PaymentsListReply{
		Payments: output,
	}
	return paymentList, nil
}

//message used to sign is of the form ("__list_unclaimed", mpe_address, current_block_number)
func (service *ProviderControlService) verifySignerForListUnclaimed(request *GetPaymentsListRequest) error {
	return service.verifySigner(service.getMessageBytes("__list_unclaimed", request), request.GetSignature())
}

func (service *ProviderControlService) getMessageBytes(prefixMessage string, request *GetPaymentsListRequest) []byte {
	message := bytes.Join([][]byte{
		[]byte(prefixMessage),
		service.serviceMetaData.GetMpeAddress().Bytes(),
		abi.U256(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	return message
}

func (service *ProviderControlService) verifySigner(message []byte, signature []byte) error {
	signer, err := authutils.GetSignerAddressFromMessage(message, signature)
	if err != nil {
		log.Error(err)
		return err
	}
	if err = authutils.VerifyAddress(*signer, service.serviceMetaData.GetPaymentAddress()); err != nil {
		return err
	}
	return nil
}

//Begin the claim process on the current channel and Increment the channel nonce and
//decrease the full amount to allow channel sender to continue working with remaining amount.
func (service *ProviderControlService) beginClaimOnChannel(channelId *big.Int) (*PaymentReply, error) {
	latestChannel, _, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: channelId})
	if err != nil {
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

//Verify if the signer is same as the payment address in metadata
//__start_claim”, mpe_address, channel_id, channel_nonce
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
		log.Error("error in retrieving claims")
		return nil, err
	}
	output := make([]*PaymentReply, 0)
	for _, claimRetrieved := range claimsRetrieved {
		payment := claimRetrieved.Payment()
		if payment.Signature == nil || payment.Amount.Int64() == 0 {
			log.Errorf("The Signature or the Amount is not defined on the Payment with"+
				" Channel Id:%v , Nonce:%v", payment.ChannelID, payment.ChannelNonce)
			continue
		}
		paymentReply := &PaymentReply{
			ChannelId:    bigIntToBytes(payment.ChannelID),
			ChannelNonce: bigIntToBytes(payment.ChannelNonce),
			SignedAmount: bigIntToBytes(payment.Amount),
			Signature:    payment.Signature,
		}
		output = append(output, paymentReply)
	}
	reply := &PaymentsListReply{
		Payments: output,
	}
	return reply, nil
}

//message used to sign is of the form ("__list_in_progress", mpe_address, current_block_number)
func (service *ProviderControlService) verifySignerForListInProgress(request *GetPaymentsListRequest) error {
	return service.verifySigner(service.getMessageBytes("__list_in_progress", request), request.GetSignature())
}

//No write operation on block chains are done by Daemon (will be take care of by the snet client )
//Finish on the claim should be called only after the payment is successfully claimed and block chain is updated accordingly.
//One way to determine this is by checking the nonce in the block chain with the nonce in the payment,
//for a given channel if the block chain nonce is greater than that of the nonce from etcd storage => that the claim is already done in block chain.
//and the Finish method is called on the claim.
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
		//Compare the state of this payment in progress with what is available in block chain
		if blockChainChannel.Nonce.Cmp(payment.ChannelNonce) > 0 {
			//if the Nonce on this block chain is higher than that of the Payment,
			//means that the payment has been completed , hence update the etcd state with this
			log.Debugf("for channel id:%v the nonce of channel from Block chain = %v is "+
				"greater than nonce of channel from etcd storage :%v , Nonce:%v",
				payment.ChannelID, blockChainChannel.Nonce, payment.ChannelNonce)
			err = claimRetrieved.Finish()
			if err != nil {
				log.Error(err)
				return err
			}
		}
	}
	return nil
}

//Check if the mpe address passed matches to what is present in the metadata.
func (service *ProviderControlService) checkMpeAddress(mpeAddress string) error {
	isSameAddress := strings.Compare(service.serviceMetaData.MpeAddress, mpeAddress) == 0
	if !isSameAddress {
		return fmt.Errorf("the mpeAddress: %s passed does not match to what has been registered", mpeAddress)
	}
	return nil
}
