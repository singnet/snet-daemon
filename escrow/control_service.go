package escrow

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/blockchain"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"strings"
)

type ProviderControlService struct {
	channelService  PaymentChannelService
	serviceMetaData *blockchain.ServiceMetadata
}

func NewProviderControlService(channelService PaymentChannelService,metaData *blockchain.ServiceMetadata) *ProviderControlService {
	return &ProviderControlService{
		channelService: channelService,
		serviceMetaData:metaData,
	}
}


/*
Get list of unclaimed payments
Verify that mpe_address is correct
Verify that actual block_number is not very different (+-5 blocks) from the current_block_number from the signature - todo
Verify that message was signed by the service provider (“payment_address” in metadata should match to the signer).
Send list of unclaimed payments
*/
func (service *ProviderControlService) GetListUnclaimed(ctx context.Context, request *GetPaymentsListRequest) (paymentReply *PaymentsListReply, err error) {
	//Get details from request

	mpeAddress := request.GetMpeAddress()


	//Check if the mpe address matches to what is there in service metadata
	if err = service.checkMpeAddress(mpeAddress).Err(); err != nil {
		return
	}

	//Check if the signer is valid
	if err = service.verifySignerForListUnclaimed(request); err != nil {
		return
	}
	return service.ListChannels()
}

func (service *ProviderControlService) ListChannels() (paymentReply *PaymentsListReply, err error) {
	//get the list of channels in progress which have some amount to be claimed.
	channels, err := service.channelService.ListChannels()
	if err != nil {
		return
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

func (service *ProviderControlService) verifySignerForListUnclaimed(request *GetPaymentsListRequest) (error) {
	//Get details from request
	currentBlock := request.GetCurrentBlock()
	signature := request.GetSignature()
	//message used to sign is of the form ("__list_unclaimed", mpe_address, current_block_number)
	message := bytes.Join([][]byte{
		[]byte ("__list_unclaimed"),
		service.serviceMetaData.GetMpeAddress().Bytes(),
		abi.U256(big.NewInt(int64(currentBlock))),
	}, nil)

	signer, err := getSignerAddressFromMessage(message, signature)

	if err != nil {
		log.Error(err)
		return err
	}
	if errorStatus := service.checkSigner(*signer); errorStatus != nil {
		return errorStatus.Err()
	}

	return nil
}

//Initialize the claim for specific channel
//Verify that signature , check if “payment_address” matches to what is there in metadata
//Increase nonce and send last payment with old nonce to the caller.
//Finish (change the state in etcd ) to reflect any claims already done on block chain but have not been reflected in the storage yet.

func (service *ProviderControlService) StartClaim(ctx context.Context, startClaim *StartClaimRequest) (paymentReply *PaymentReply, err error) {
	//Get details from request
	channelId := bytesToBigInt(startClaim.GetChannelId())


	//Check if the mpe address matches to what is there in service metadata
	mpeAddress := startClaim.MpeAddress
	if err = service.checkMpeAddress(mpeAddress).Err(); err != nil {
		return
	}

	//Verify signature , check if “payment_address” matches to what is there in metadata
	err = service.verifySignerForStartClaim(startClaim)
	if err != nil {
		return
	}

	//Remove any payments already claimed on block chain
	err = service.removeClaimedPayments()
	if err != nil {
		log.Error("unable to remove payments from etcd storage which are already claimed in block chain")
		return
	}

	return service.beginClaimOnChannel(channelId)
}


//Begin the claim process on the current channel and Increment the channel nonce and
//decrease the full amount to allow channel sender to continue working with remaining amount.

func (service *ProviderControlService) beginClaimOnChannel (channelId *big.Int) (paymentReply *PaymentReply, err error) {

	latestChannel, _, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: channelId})

	if err != nil {
		return
	}
	//Check if there is any Authorized amount to initiate a claim
	if latestChannel.AuthorizedAmount.Int64() == 0  {
		err = fmt.Errorf("authorized amount is zero , hence nothing to claim on the channel Id: %v",channelId)
		return
	}

	claim, err := service.channelService.StartClaim(&PaymentChannelKey{ID: channelId}, IncrementChannelNonce)
	if err != nil {
		return
	}
	paymentReply = &PaymentReply{
		ChannelId:    bigIntToBytes(channelId),
		ChannelNonce: bigIntToBytes(claim.Payment().ChannelNonce),
	}
	return paymentReply,nil
}
//Verify if the signer is same as the payment address in metadata
//__start_claim”, mpe_address, channel_id, channel_nonce
func (service *ProviderControlService) verifySignerForStartClaim(startClaim *StartClaimRequest) error {
	channelId := bytesToBigInt(startClaim.GetChannelId())

	signature := startClaim.Signature


	latestChannel, ok, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: channelId})
	if !ok || err != nil {
		log.WithFields(log.Fields{
			"payment channel retrieval error": err,
		}).Errorf("unable to retrieve latest channel state from block chain for channel Id:", channelId)
		return err
	}
	message := bytes.Join([][]byte{
		[]byte ("__start_claim"),
		service.serviceMetaData.GetMpeAddress().Bytes(),
		bigIntToBytes(channelId),
		bigIntToBytes(latestChannel.Nonce),
	}, nil)

	signer, err := getSignerAddressFromMessage(message, signature)

	if err != nil {
		log.Error(err)
		//return err
	}
	if errorStatus := service.checkSigner(*signer); errorStatus != nil {
		return errorStatus.Err()
	}

	return nil
}

//To authenticate sender request should also contain correct signature of the channel id.
//Before sending list of payments, daemon should remove all payments
//with nonce < block chain nonce from payment storage (call finalize on them?).
//It means that daemon remove all payments which were claimed already.

func (service *ProviderControlService) GetListInProgress(ctx context.Context, request *GetPaymentsListRequest) (reply *PaymentsListReply, err error) {

	if err = service.checkMpeAddress(request.GetMpeAddress()).Err(); err != nil {
		return
	}

	if err = service.verifySignerForListInProgress(request); err != nil {
		return
	}

	err = service.removeClaimedPayments()
	if err != nil {
		log.Errorf("unable to remove payments from which are already claimed")
		return
	}

	return service.ListClaims()
}

func (service *ProviderControlService) ListClaims()  (reply *PaymentsListReply, err error) {
	//retrieve all the claims in progress
	claimsRetrieved, err := service.channelService.ListClaims()
	if err != nil {
		log.Error("error in retrieving claims")
		return
	}

	output := make([]*PaymentReply, 0)

	for _, claimRetrieved := range claimsRetrieved {
		payment := claimRetrieved.Payment()
		if payment.Signature == nil || payment.Amount.Int64() == 0 {
			log.Errorf("The Signature or the Amount is not defined on the Payment with" +
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
	reply = &PaymentsListReply{
		Payments: output,
	}
	return
}

func (service *ProviderControlService) verifySignerForListInProgress(request *GetPaymentsListRequest) (error) {
	currentBlock := request.GetCurrentBlock()
	signature := request.GetSignature()
	message := bytes.Join([][]byte{
		[]byte ("__list_in_progress"),
		service.serviceMetaData.GetMpeAddress().Bytes(),
		abi.U256(big.NewInt(int64(currentBlock))),
	}, nil)

	signer, err := getSignerAddressFromMessage(message, signature)

	if err != nil {
		log.Error(err)
		return err
	}
	if errorStatus := service.checkSigner(*signer); errorStatus != nil {
		return errorStatus.Err()
	}

	return nil
}

//No write operation on block chains are done by Daemon (will be take care of by the snet client )
//Finish should be called only after the payment is successfully claimed and block chain is updated accordingly.
//One way to determine this is by checking the nonce in the block chain with the nonce in the payment,
//if the block chain has a greater nonce => that the claim is already done in block chain.
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
			log.WithFields(log.Fields{
				"Payment channel retrieval error": err,
			}).Errorf("unable to retrieve channel state from block chain for Payment %v", payment)
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
			}
			continue
		}
	}
	return nil
}

//Check if the MPE Address passed is the same
func (service *ProviderControlService) checkSigner(address common.Address) (*status.Status) {
	isSameAddress := service.serviceMetaData.GetPaymentAddress() == address
	//Check if the payment address passed matches to what has been received.
	errorStatus := status.New(codes.InvalidArgument,
		fmt.Errorf("The payment Address : %s  does not match to what has been registered", address).Error())
	if !isSameAddress {
		return errorStatus
	}
	return nil
}

//Check if the MPE Address passed is the same
func (service *ProviderControlService) checkMpeAddress(mpeAddress string) (*status.Status) {
	isSameAddress := strings.Compare(service.serviceMetaData.MpeAddress, mpeAddress) == 0
	//Check if the mpe address passed matches to what is present in the metadata.
	errorStatus := status.New(codes.InvalidArgument,
		fmt.Errorf("mpeAddress : %s passed does not match to what has been registered", mpeAddress).Error())
	if !isSameAddress {
		log.WithFields(log.Fields{
			"mpeAddress": mpeAddress,
		}).Error(errorStatus)
		return errorStatus
	}
	return nil
}
