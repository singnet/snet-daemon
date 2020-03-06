package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
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
	md.Set(handler.AllowedUserSignatureHeader, string(SignAllowedUserSignature(allowedUserPvtKey)))
	md.Set(handler.PaymentTypeHeader, AllowedUserPaymentType)
	cntxt := &handler.GrpcStreamContext{
		MD: md,
	}
	_, err := testhandler.Payment(cntxt)
	assert.Nil(t, err)

}

func SignAllowedUserSignature(privateKey *ecdsa.PrivateKey) []byte {
	message := bytes.Join([][]byte{
		[]byte(AllowedUserPrefixSignature),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
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
