package freecall

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/errs"
	"github.com/singnet/snet-daemon/v6/utils"
	"go.uber.org/zap"
)

const (
	PrefixSignature        = "__prefix_free_trial"
	TokenLifetime          = 172800 // in blocks
	allowedBlockDifference = 5
)

type PaymentValidator struct {
	currentBlock                   func() (currentBlock *big.Int, err error)
	freeCallSignerAddress          common.Address
	trustedFreeCallSignerAddresses []common.Address
	freeCallSigner                 *ecdsa.PrivateKey
}

func NewPaymentValidator(funcCurrentBlock func() (currentBlock *big.Int, err error), signerAddress common.Address, signer *ecdsa.PrivateKey, trustedAddresses []common.Address) *PaymentValidator {
	return &PaymentValidator{
		currentBlock:                   funcCurrentBlock,
		freeCallSignerAddress:          signerAddress,
		trustedFreeCallSignerAddresses: trustedAddresses,
		freeCallSigner:                 signer,
	}
}

func (validator *PaymentValidator) NewToken(userAddress string, userID *string, tokenLifetimeBlocks *uint64) ([]byte, *big.Int) {

	userAddr := common.HexToAddress(userAddress)

	latestBlockNumber, err := validator.currentBlock()
	if err != nil {
		return nil, nil
	}

	blockExpiration := big.NewInt(TokenLifetime)
	if tokenLifetimeBlocks != nil && *tokenLifetimeBlocks <= TokenLifetime {
		blockExpiration.SetUint64(*tokenLifetimeBlocks)
	}

	deadlineBlockOfToken := latestBlockNumber.Add(latestBlockNumber, blockExpiration)

	message := BuildTokenMessage(&userAddr, deadlineBlockOfToken, userID)

	signedToken := utils.GetSignature(message, validator.freeCallSigner)
	signedToken = append(signedToken, []byte("_"+deadlineBlockOfToken.String())...)
	return signedToken, deadlineBlockOfToken
}

func ParseToken(token []byte) (sig []byte, block *big.Int, err error) {
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
		[]byte(PrefixSignature),
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

func (validator *PaymentValidator) CompareSignerAddrs(addr common.Address) error {
	if slices.ContainsFunc(validator.trustedFreeCallSignerAddresses, func(address common.Address) bool {
		return addr == address
	}) {
		return nil
	}

	if addr == validator.freeCallSignerAddress {
		return nil
	}
	return errs.NewPaymentError(errs.Unauthenticated, "payment signer %v is not valid", addr.Hex())
}

func (validator *PaymentValidator) Validate(payment *Payment) (err error) {
	tokenSignerAddress, err := validator.getSignerOfAuthTokenForFreeCall(payment)
	if err != nil {
		return errs.NewPaymentError(errs.Unauthenticated, "sign is not valid: %v", err)
	}

	// check that free call token signed by daemon
	if *tokenSignerAddress != validator.freeCallSignerAddress {
		return errs.NewPaymentError(errs.Unauthenticated, "token sign is not valid")
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

// Check if the block number passed is not more +- 5 from the latest block number on chain
func (validator *PaymentValidator) compareWithLatestBlockNumber(blockNumberPassed *big.Int) error {
	latestBlockNumber, err := validator.currentBlock()
	if err != nil {
		return err
	}
	differenceInBlockNumber := blockNumberPassed.Sub(blockNumberPassed, latestBlockNumber)
	if differenceInBlockNumber.Abs(differenceInBlockNumber).Uint64() > allowedBlockDifference {
		return fmt.Errorf("authentication failed as the signature passed has expired")
	}
	return nil
}

func (validator *PaymentValidator) CheckIfBlockExpired(expiredBlock *big.Int) error {
	currentBlockNumber, err := validator.currentBlock()
	if err != nil {
		return err
	}

	if expiredBlock.Cmp(currentBlockNumber) < 0 {
		return fmt.Errorf("authentication failed as the Free Call Token passed has expired")
	}
	return nil
}

func BuildTokenMessage(addr *common.Address, expirationBlock *big.Int, userID *string) (token []byte) {
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

func (validator *PaymentValidator) getSignerOfAuthTokenForFreeCall(payment *Payment) (signer *common.Address, err error) {
	signer, err = getAddressFromSigForFreeCall(payment)
	if err != nil {
		return nil, err
	}
	// checking that address in the message is the same that in metadata
	if common.HexToAddress(payment.Address) != *signer {
		return nil, fmt.Errorf("unauthorized signer: %v not equal %v, maybe invalid signature struct", payment.Address, signer.Hex())
	}

	//zap.L().Debug("Signer of request will be passed to token", zap.String("address", signer.Hex()))

	message := BuildTokenMessage(signer, payment.AuthTokenExpiryBlockNumber, &payment.UserID)
	return utils.GetSignerAddressFromMessage(message, payment.AuthTokenParsed)
}

func getAddressFromSigForFreeCall(payment *Payment) (signer *common.Address, err error) {

	message := bytes.Join([][]byte{
		[]byte(PrefixSignature),
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
