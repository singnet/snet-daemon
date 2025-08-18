package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/utils"

	"go.uber.org/zap"
)

const (
	PrefixInSignature = "__MPE_claim_message"
	//Agreed constant value
	FreeCallPrefixSignature = "__prefix_free_trial"
	//Agreed constant value
	AllowedUserPrefixSignature = "__authorized_user"

	FreeCallTokenLifetime = 172800 // in blocks
)

type FreeCallPaymentValidator struct {
	currentBlock                   func() (currentBlock *big.Int, err error)
	freeCallSignerAddress          common.Address
	trustedFreeCallSignerAddresses []common.Address
	freeCallSigner                 *ecdsa.PrivateKey
}

func NewFreeCallPaymentValidator(funcCurrentBlock func() (currentBlock *big.Int, err error), signerAddress common.Address, signer *ecdsa.PrivateKey, trustedAddresses []common.Address) *FreeCallPaymentValidator {
	return &FreeCallPaymentValidator{
		currentBlock:                   funcCurrentBlock,
		freeCallSignerAddress:          signerAddress,
		trustedFreeCallSignerAddresses: trustedAddresses,
		freeCallSigner:                 signer,
	}
}

func (validator *FreeCallPaymentValidator) NewFreeCallToken(userAddress string, userID *string, tokenLifetimeBlocks *uint64) ([]byte, *big.Int) {

	userAddr := common.HexToAddress(userAddress)

	latestBlockNumber, err := validator.currentBlock()
	if err != nil {
		return nil, nil
	}

	blockExpiration := big.NewInt(FreeCallTokenLifetime)
	if tokenLifetimeBlocks != nil && *tokenLifetimeBlocks <= FreeCallTokenLifetime {
		blockExpiration.SetUint64(*tokenLifetimeBlocks)
	}

	deadlineBlockOfToken := latestBlockNumber.Add(latestBlockNumber, blockExpiration)

	message := BuildFreeCallTokenStruct(&userAddr, deadlineBlockOfToken, userID)

	signedToken := utils.GetSignature(message, validator.freeCallSigner)
	signedToken = append(signedToken, []byte("_"+deadlineBlockOfToken.String())...)
	return signedToken, deadlineBlockOfToken
}

func ParseFreeCallToken(token []byte) (sig []byte, block *big.Int, err error) {
	i := bytes.LastIndexByte(token, '_')
	if i == -1 {
		return nil, nil, errors.New("no '_' found")
	}

	sig = token[:i]
	block = new(big.Int)
	if _, ok := block.SetString(string(token[i+1:]), 10); !ok {
		return nil, nil, errors.New("invalid block number")
	}

	return sig, block, nil
}

func getAddressFromSignatureForNewFreeCallToken(request *GetFreeCallTokenRequest, groupID string) (signer *common.Address, err error) {

	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(request.GetAddress()),
		[]byte(request.GetUserId()),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		[]byte(groupID),
		bigIntToBytes(big.NewInt(int64(request.GetCurrentBlock()))),
	}, nil)

	signer, err = utils.GetSignerAddressFromMessage(message, request.Signature)
	if err != nil {
		zap.L().Error("Cannot get signer from message", zap.Error(err))
		return nil, err
	}
	if err = checkCurationValidations(signer); err != nil {
		zap.L().Error(err.Error())
		return nil, err
	}
	return signer, err
}

type AllowedUserPaymentValidator struct {
}

func (validator *AllowedUserPaymentValidator) Validate(payment *Payment) (err error) {
	_, err = getSignerAddressFromPayment(payment)
	return err
}

func (validator *FreeCallPaymentValidator) CompareSignerAddrs(addr common.Address) error {
	if slices.ContainsFunc(validator.trustedFreeCallSignerAddresses, func(address common.Address) bool {
		return addr == address
	}) {
		return nil
	}

	if addr == validator.freeCallSignerAddress {
		return nil
	}
	return NewPaymentError(Unauthenticated, "payment signer %v is not valid", addr.Hex())
}

func (validator *FreeCallPaymentValidator) Validate(payment *FreeCallPayment) (err error) {
	tokenSignerAddress, err := validator.getSignerOfAuthTokenForFreeCall(payment)
	if err != nil {
		return NewPaymentError(Unauthenticated, "sign is not valid: %v", err)
	}

	// check that free call token signed by daemon
	if *tokenSignerAddress != validator.freeCallSignerAddress {
		return NewPaymentError(Unauthenticated, "token sign is not valid")
	}

	if err := validator.CheckIfBlockExpired(payment.AuthTokenExpiryBlockNumber); err != nil {
		return err
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

// NewChannelPaymentValidator returns a new payment validator instance
func NewChannelPaymentValidator(processor blockchain.Processor, metadata *blockchain.OrganizationMetaData) *ChannelPaymentValidator {
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

	signerAddressFieldLog := zap.String("signerAddress", utils.AddressToHex(signerAddress))
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
	if differenceInBlockNumber.Abs(differenceInBlockNumber).Uint64() > AllowedBlockDifference {
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

func BuildFreeCallTokenStruct(addr *common.Address, expirationBlock *big.Int, userID *string) (token []byte) {
	if userID == nil {
		userID = new(string)
	}

	message := bytes.Join([][]byte{
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.DaemonGroupName)),
		addr.Bytes(), // user or market app address
		[]byte(*userID),
		bigIntToBytes(expirationBlock),
	}, nil)

	return message
}

func (validator *FreeCallPaymentValidator) getSignerOfAuthTokenForFreeCall(payment *FreeCallPayment) (signer *common.Address, err error) {
	signer, err = getAddressFromSigForFreeCall(payment)
	if err != nil {
		return nil, err
	}
	// checking that address in the message is the same that in metadata
	if common.HexToAddress(payment.Address) != *signer {
		return nil, fmt.Errorf("unauthorized signer: %v not equal %v, maybe invalid signature struct", payment.Address, signer.Hex())
	}

	//zap.L().Debug("Signer of request will be passed to token", zap.String("address", signer.Hex()))

	message := BuildFreeCallTokenStruct(signer, payment.AuthTokenExpiryBlockNumber, &payment.UserID)
	return utils.GetSignerAddressFromMessage(message, payment.AuthTokenParsed)
}

func getAddressFromSigForFreeCall(payment *FreeCallPayment) (signer *common.Address, err error) {

	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(payment.Address),
		[]byte(payment.UserID),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		[]byte(payment.GroupId),
		bigIntToBytes(payment.CurrentBlockNumber),
		payment.AuthToken,
	}, nil)

	signer, err = utils.GetSignerAddressFromMessage(message, payment.Signature)
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
	//hit by unknown users during a curation process
	if config.GetBool(config.AllowedUserFlag) {
		if !config.IsAllowedUser(signer) {
			return fmt.Errorf("you are not Authorized to call this service during curation process")
		}
	}
	return nil
}

func getSignerAddressFromPayment(payment *Payment) (signer *common.Address, err error) {
	message := bytes.Join([][]byte{
		[]byte(PrefixInSignature),
		payment.MpeContractAddress.Bytes(),
		bigIntToBytes(payment.ChannelID),
		bigIntToBytes(payment.ChannelNonce),
		bigIntToBytes(payment.Amount),
	}, nil)

	signer, err = utils.GetSignerAddressFromMessage(message, payment.Signature)
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
