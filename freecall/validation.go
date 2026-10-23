package freecall

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
	"github.com/singnet/snet-daemon/v6/errs"
	"github.com/singnet/snet-daemon/v6/utils"
	"go.uber.org/zap"
)

const (
	// PrefixSignature identifies caller-signed free-call requests. These are
	// separate from MPE claim authorizations and their __MPE_claim_message prefix.
	PrefixSignature = "__prefix_free_trial"
	// TokenLifetime is the default and maximum accepted custom lifetime in blocks.
	// Larger requested values fall back to this default rather than being rejected.
	TokenLifetime = 172800
	// allowedBlockDifference is the inclusive request-block freshness window.
	allowedBlockDifference = 5
)

// PaymentValidator authenticates the caller and provider token, checks token
// expiration, and validates request-block freshness. It does not check MPE channel
// state, wallet balance, or the remaining local free-call allowance.
type PaymentValidator struct {
	blockValidator                 *blockchain.EthereumBlockValidator
	freeCallSignerAddress          common.Address
	trustedFreeCallSignerAddresses []common.Address
	freeCallSigner                 *ecdsa.PrivateKey
}

// NewPaymentValidator uses the service group's metadata signer as the token
// authority. signer is needed for NewToken and may be nil for validation-only
// handlers. trustedAddresses controls backend eligibility at token issuance;
// it does not allow those backends to sign provider-issued tokens themselves.
func NewPaymentValidator(funcCurrentBlock func() (currentBlock *big.Int, err error), signerAddress common.Address, signer *ecdsa.PrivateKey, trustedAddresses []common.Address) *PaymentValidator {
	return &PaymentValidator{
		blockValidator:                 blockchain.NewEthereumBlockValidator(funcCurrentBlock),
		freeCallSignerAddress:          signerAddress,
		trustedFreeCallSignerAddresses: trustedAddresses,
		freeCallSigner:                 signer,
	}
}

// NewToken signs a free-call token expiring at the daemon's current block plus
// the resolved lifetime. A nil or oversized lifetime uses TokenLifetime; zero
// makes the token valid through the current block. StateService checks request
// authentication and eligibility separately. A block-read failure returns nil, nil.
func (validator *PaymentValidator) NewToken(userAddress string, userID *string, tokenLifetimeBlocks *uint64) ([]byte, *big.Int) {

	userAddr := common.HexToAddress(userAddress)

	latestBlockNumber, err := validator.blockValidator.CurrentBlock()
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

// ParseToken splits the full token at its last underscore into raw signature
// bytes and a decimal expiration block. Signature bytes may contain underscores.
// Parsing alone does not authenticate the signature or check its expiration.
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

// getAddressFromSignatureForNewFreeCallToken recovers the token-request signer.
// Address and identifiers are exact UTF-8 strings, while current_block uses
// 32-byte big-endian encoding. groupID is the metadata group ID, not its name.
// The requested token lifetime is not included in this signed payload.
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

// CompareSignerAddrs accepts a trusted backend or the configured token signer.
// Token validation is stricter: only freeCallSignerAddress may issue a token.
// StateService separately checks trusted backend membership for user-ID issuance.
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

	// Only the free_call_signer_address from service metadata may sign tokens.
	// A trusted backend signs caller requests, not the provider's authorization.
	if *tokenSignerAddress != validator.freeCallSignerAddress {
		return errs.NewPaymentError(errs.Unauthenticated, "token sign is not valid")
	}

	if err := validator.CheckIfBlockExpired(payment.AuthTokenExpiryBlockNumber); err != nil {
		return err
	}

	// Request freshness is independent of the token's inclusive expiration block.
	if err := validator.compareWithLatestBlockNumber(payment.CurrentBlockNumber); err != nil {
		return err
	}

	return nil
}

// compareWithLatestBlockNumber accepts an absolute block difference of at most
// five, including both boundaries. Missing or negative request blocks are invalid.
func (validator *PaymentValidator) compareWithLatestBlockNumber(blockNumberPassed *big.Int) error {
	return validator.blockValidator.ValidateBlock(blockNumberPassed, allowedBlockDifference)
}

// CheckIfBlockExpired rejects tokens only after their expiration block. Equality
// is valid; the request freshness tolerance does not extend the token's lifetime.
func (validator *PaymentValidator) CheckIfBlockExpired(expiredBlock *big.Int) error {
	currentBlockNumber, err := validator.blockValidator.CurrentBlock()
	if err != nil {
		return err
	}

	if expiredBlock.Cmp(currentBlockNumber) < 0 {
		return fmt.Errorf("authentication failed as the Free Call Token passed has expired")
	}
	return nil
}

// BuildTokenMessage builds the provider-signed payload from the configured
// organization ID and daemon group name, a raw 20-byte caller address, optional
// user-ID text, and a 32-byte expiration block. Strings are concatenated without
// separators or lengths. Unlike caller requests, it omits the service ID and
// metadata group ID; it also contains no MPE or Registry address.
// A nil userID is encoded as an empty string.
func BuildTokenMessage(addr *common.Address, expirationBlock *big.Int, userID *string) (token []byte) {
	if userID == nil {
		userID = new(string)
	}

	message := bytes.Join([][]byte{
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.DaemonGroupName)),
		addr.Bytes(), // Raw wallet/backend address, unlike the address text in caller requests.
		[]byte(*userID),
		bigIntToBytes(expirationBlock),
	}, nil)

	return message
}

// getSignerOfAuthTokenForFreeCall first authenticates the caller's signed request,
// then recovers the provider signature over that caller's token payload.
func (validator *PaymentValidator) getSignerOfAuthTokenForFreeCall(payment *Payment) (signer *common.Address, err error) {
	signer, err = getAddressFromSigForFreeCall(payment)
	if err != nil {
		return nil, err
	}
	// Require the recovered caller to match the address supplied in the request.
	if common.HexToAddress(payment.Address) != *signer {
		return nil, fmt.Errorf("unauthorized signer: %v not equal %v, maybe invalid signature struct", payment.Address, signer.Hex())
	}

	message := BuildTokenMessage(signer, payment.AuthTokenExpiryBlockNumber, &payment.UserID)
	return utils.GetSignerAddressFromMessage(message, payment.AuthTokenParsed)
}

// getAddressFromSigForFreeCall authenticates an availability request or service
// call. Its payload extends the token-request format with the full raw token,
// including its expiration suffix, and applies the optional curation allowlist.
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
	// Apply the optional testing/curation allowlist to the caller, independently
	// of trusted backend membership and the service's token-signing address.
	if config.GetBool(config.AllowedUserFlag) {
		if !config.IsAllowedUser(signer) {
			return fmt.Errorf("you are not Authorized to call this service during curation process")
		}
	}
	return nil
}
