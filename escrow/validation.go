package escrow

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v5/authutils"
	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	PrefixInSignature = "__MPE_claim_message"
	//Agreed constant value
	FreeCallPrefixSignature = "__prefix_free_trial"
	//Agreed constant value
	AllowedUserPrefixSignature = "__authorized_user"
)

type FreeCallPaymentValidator struct {
	currentBlock   func() (currentBlock *big.Int, err error)
	freeCallSigner common.Address
}

func NewFreeCallPaymentValidator(funcCurrentBlock func() (currentBlock *big.Int, err error), signer common.Address) *FreeCallPaymentValidator {
	return &FreeCallPaymentValidator{
		currentBlock:   funcCurrentBlock,
		freeCallSigner: signer,
	}
}

type AllowedUserPaymentValidator struct {
}

func (validator *AllowedUserPaymentValidator) Validate(payment *Payment) (err error) {
	_, err = getSignerAddressFromPayment(payment)
	return err
}

func (validator *FreeCallPaymentValidator) Validate(payment *FreeCallPayment) (err error) {
	newSignature := true //this will be removed once dapp makes the changes to move to new Signature
	signerAddress, err := validator.getSignerOfAuthTokenForFreeCall(payment)
	if err != nil || *signerAddress != validator.freeCallSigner {
		//Make sure the current Dapp is backward compatible, this will be removed once Dapp
		//Makes the latest signature change with Token for Free calls
		if signerAddress, err = validator.getSignerAddressForFreeCall(payment); err != nil {
			return NewPaymentError(Unauthenticated, "payment signature is not valid")
		}
		newSignature = false
	}
	if *signerAddress != validator.freeCallSigner {
		return NewPaymentError(Unauthenticated, "payment signer is not valid %v , %v", signerAddress.Hex(), validator.freeCallSigner.Hex())
	}
	if newSignature {
		if err := validator.CheckIfBlockExpired(payment.AuthTokenExpiryBlockNumber); err != nil {
			return err
		}
	}
	//Check for the current block Number
	if err := validator.compareWithLatestBlockNumber(payment.CurrentBlockNumber); err != nil {
		return err
	}

	return nil
}

// ChannelPaymentValidator validates payment using payment channel state.
type ChannelPaymentValidator struct {
	currentBlock               func() (currentBlock *big.Int, err error)
	paymentExpirationThreshold func() (threshold *big.Int)
}

// NewChannelPaymentValidator returns new payment validator instance
func NewChannelPaymentValidator(processor *blockchain.Processor, cfg *viper.Viper, metadata *blockchain.OrganizationMetaData) *ChannelPaymentValidator {
	return &ChannelPaymentValidator{
		currentBlock: processor.CurrentBlock,
		paymentExpirationThreshold: func() *big.Int {
			return metadata.GetPaymentExpirationThreshold()
		},
	}
}

// Validate returns instance of PaymentError as error if validation fails, nil
// otherwise.
func (validator *ChannelPaymentValidator) Validate(payment *Payment, channel *PaymentChannelData) (err error) {
	paymentFieldLog := zap.Any("payment", payment)
	channelFieldLog := zap.Any("channel", channel)

	if payment.ChannelNonce.Cmp(channel.Nonce) != 0 {
		zap.L().Warn("Incorrect nonce is sent by client", paymentFieldLog, channelFieldLog)
		return NewPaymentError(IncorrectNonce, "incorrect payment channel nonce, latest: %v, sent: %v", channel.Nonce, payment.ChannelNonce)
	}

	signerAddress, err := getSignerAddressFromPayment(payment)
	if err != nil {
		return NewPaymentError(Unauthenticated, "payment signature is not valid")
	}

	signerAddressFieldLog := zap.String("signerAddress", blockchain.AddressToHex(signerAddress))
	if *signerAddress != channel.Signer && *signerAddress != channel.Sender {
		zap.L().Warn("Channel signer is not equal to payment signer/sender", signerAddressFieldLog)
		return NewPaymentError(Unauthenticated, "payment is not signed by channel signer/sender")
	}
	currentBlock, e := validator.currentBlock()
	if e != nil {
		return NewPaymentError(Internal, "cannot determine current block")
	}
	expirationThreshold := validator.paymentExpirationThreshold()
	currentBlockWithThreshold := new(big.Int).Add(currentBlock, expirationThreshold)
	if currentBlockWithThreshold.Cmp(channel.Expiration) >= 0 {
		zap.L().Warn("Channel expiration time is after expiration threshold", zap.Any("currentBlock", currentBlock), zap.Any("expirationThreshold", expirationThreshold))
		return NewPaymentError(Unauthenticated, "payment channel is near to be expired, expiration time: %v, current block: %v, expiration threshold: %v", channel.Expiration, currentBlock, expirationThreshold)
	}

	if channel.FullAmount.Cmp(payment.Amount) < 0 {
		zap.L().Warn("Not enough tokens on payment channel")
		return NewPaymentError(Unauthenticated, "not enough tokens on payment channel, channel amount: %v, payment amount: %v", channel.FullAmount, payment.Amount)
	}

	return
}

// Check if the block number passed is not more +- 5 from the latest block number on chain
func (validator *FreeCallPaymentValidator) compareWithLatestBlockNumber(blockNumberPassed *big.Int) error {
	latestBlockNumber, err := validator.currentBlock()
	if err != nil {
		return err
	}
	differenceInBlockNumber := blockNumberPassed.Sub(blockNumberPassed, latestBlockNumber)
	if differenceInBlockNumber.Abs(differenceInBlockNumber).Uint64() > authutils.AllowedBlockChainDifference {
		return fmt.Errorf("authentication failed as the signature passed has expired")
	}
	return nil
}

func (validator *FreeCallPaymentValidator) CheckIfBlockExpired(expiredBlock *big.Int) error {
	currentBlockNumber, err := validator.currentBlock()
	if err != nil {
		return err
	}

	if expiredBlock.Cmp(currentBlockNumber) < 0 {
		return fmt.Errorf("authentication failed as the Free Call Token passed has expired")
	}
	return nil
}

// deprecated
func (validator *FreeCallPaymentValidator) getSignerAddressForFreeCall(payment *FreeCallPayment) (signer *common.Address, err error) {

	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(payment.UserId),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		bigIntToBytes(payment.CurrentBlockNumber),
	}, nil)

	signer, err = authutils.GetSignerAddressFromMessage(message, payment.Signature)
	if err != nil {
		zap.L().Error("Cannot get signer from payment", zap.Any("payment", payment), zap.Error(err))
		return nil, err
	}
	return signer, err
}

func getSignerAddressFromPayment(payment *Payment) (signer *common.Address, err error) {
	message := bytes.Join([][]byte{
		[]byte(PrefixInSignature),
		payment.MpeContractAddress.Bytes(),
		bigIntToBytes(payment.ChannelID),
		bigIntToBytes(payment.ChannelNonce),
		bigIntToBytes(payment.Amount),
	}, nil)

	signer, err = authutils.GetSignerAddressFromMessage(message, payment.Signature)
	if err != nil {
		zap.L().Error("Cannot get signer from payment", zap.Error(err), zap.Any("payment", payment))
		return nil, err
	}
	if err = checkCurationValidations(signer); err != nil {
		zap.L().Error(err.Error())
		return nil, err
	}

	return signer, err
}

func (validator *FreeCallPaymentValidator) getSignerOfAuthTokenForFreeCall(payment *FreeCallPayment) (signer *common.Address, err error) {
	//signer-token = (user@mail, user-public-key, token_issue_date), this is generated by Marketplace Dapp
	signer, err = getUserAddressFromSignatureOfFreeCalls(payment)
	if err != nil {
		return nil, err
	}
	zap.L().Debug("signer of req - will be passed to token: ", zap.String("signer", signer.Hex()))
	zap.L().Debug("AuthTokenExpiryBlockNumber", zap.Int64("value", payment.AuthTokenExpiryBlockNumber.Int64()))
	message := bytes.Join([][]byte{
		[]byte(payment.UserId),
		signer.Bytes(), // user address
		bigIntToBytes(payment.AuthTokenExpiryBlockNumber),
	}, nil)
	return authutils.GetSignerAddressFromMessage(message, payment.AuthToken)

}

// user signs using his private key, the public address of this user should be in the token issued by Dapp
func getUserAddressFromSignatureOfFreeCalls(payment *FreeCallPayment) (signer *common.Address, err error) {
	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(payment.UserId),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		[]byte(payment.GroupId),
		bigIntToBytes(payment.CurrentBlockNumber),
		payment.AuthToken,
	}, nil)

	signer, err = authutils.GetSignerAddressFromMessage(message, payment.Signature)
	if err != nil {
		zap.L().Error("Cannot get signer from payment", zap.Error(err), zap.Any("payment", payment))
		return nil, err
	}
	if err = checkCurationValidations(signer); err != nil {
		zap.L().Error(err.Error())
		return nil, err
	}
	return signer, err
}

func bigIntToBytes(value *big.Int) []byte {
	return common.BigToHash(value).Bytes()
}

func bytesToBigInt(bytes []byte) *big.Int {
	return (&big.Int{}).SetBytes(bytes)
}

func checkCurationValidations(signer *common.Address) error {
	//This is only to protect the Service provider in test environment from being
	//hit by unknown users during curation process
	if config.GetBool(config.AllowedUserFlag) {
		if !config.IsAllowedUser(signer) {
			return fmt.Errorf("you are not Authorized to call this service during curation process")
		}
	}
	return nil
}
