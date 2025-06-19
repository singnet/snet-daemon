package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"math/big"
	"testing"
)

func TestAllowedUserPaymentHandler(t *testing.T) {

}

func Test_allowedUserPaymentHandler_Type(t *testing.T) {

}

func Test_allowedUserPaymentHandler_Payment(t *testing.T) {

}

func Test_allowedUserPaymentHandler_getPaymentFromContext(t *testing.T) {
	config.Vip().Set(config.AllowedUserFlag, true)
	allowedUserPvtKey := GenerateTestPrivateKey()
	allowedUser := crypto.PubkeyToAddress(allowedUserPvtKey.PublicKey)
	config.Vip().Set(config.AllowedUserAddresses, []string{allowedUser.Hex()})
	config.SetAllowedUsers()
	testhandler := AllowedUserPaymentHandler()
	md := metadata.New(map[string]string{})
	md.Set(handler.PaymentChannelSignatureHeader, string(SignAllowedUserSignature(allowedUserPvtKey)))
	md.Set(handler.PaymentTypeHeader, EscrowPaymentType)
	md.Set(handler.PaymentChannelAmountHeader, "3")
	md.Set(handler.PaymentChannelNonceHeader, "0")
	md.Set(handler.PaymentChannelIDHeader, "1")
	cntxt := &handler.GrpcStreamContext{
		MD: md,
	}
	_, err := testhandler.Payment(cntxt)
	assert.Equal(t, "rpc error: code = InvalidArgument desc = missing \"snet-payment-mpe-address\"", err.Err().Error())
	md.Set(handler.PaymentMultiPartyEscrowAddressHeader, "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207")
	_, err = testhandler.Payment(cntxt)
	assert.Nil(t, err)

}

func SignAllowedUserSignature(privateKey *ecdsa.PrivateKey) []byte {
	message := bytes.Join([][]byte{
		[]byte(PrefixInSignature),
		common.HexToAddress("0x94d04332C4f5273feF69c4a52D24f42a3aF1F207").Bytes(),
		bigIntToBytes(big.NewInt(1)),
		bigIntToBytes(big.NewInt(0)),
		bigIntToBytes(big.NewInt(3)),
	}, nil)

	return getSignature(message, privateKey)
}

func Test_allowedUserPaymentHandler_Complete(t *testing.T) {
	handler := AllowedUserPaymentHandler()
	assert.Nil(t, handler.Complete(nil))
}

func Test_allowedUserPaymentHandler_CompleteAfterError(t *testing.T) {
	handler := AllowedUserPaymentHandler()
	assert.Nil(t, handler.CompleteAfterError(nil, nil))

}
