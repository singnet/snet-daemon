package freecall

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type authFixture struct {
	validator *PaymentValidator
	signerKey *ecdsa.PrivateKey
	userKey   *ecdsa.PrivateKey
	org       *blockchain.OrganizationMetaData
	metadata  *blockchain.ServiceMetadata
}

func (f *authFixture) groupID() ([32]byte, error) {
	return f.org.GetGroupId(), nil
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	previous := config.Vip()
	config.SetVip(viper.New())
	t.Cleanup(func() { config.SetVip(previous) })
	config.Vip().Set(config.OrganizationId, "test-org")
	config.Vip().Set(config.ServiceId, "test-service")
	config.Vip().Set(config.DaemonGroupName, "default_group")

	signerKey := GenerateTestPrivateKey()
	signer := crypto.PubkeyToAddress(signerKey.PublicKey)
	org, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	require.NoError(t, err)
	metadataJSON := strings.ReplaceAll(coverageMetadataJson, "0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F", signer.Hex())
	metadata, err := blockchain.InitServiceMetaDataFromJson([]byte(metadataJSON))
	require.NoError(t, err)
	return &authFixture{
		validator: NewPaymentValidator(func() (*big.Int, error) { return big.NewInt(100), nil }, signer, signerKey, nil),
		signerKey: signerKey,
		userKey:   GenerateTestPrivateKey(),
		org:       org,
		metadata:  metadata,
	}
}

func (f *authFixture) payment(t *testing.T) *Payment {
	t.Helper()
	payment := &Payment{
		Address:            crypto.PubkeyToAddress(f.userKey.PublicKey).Hex(),
		UserID:             "test-user",
		OrganizationId:     config.GetString(config.OrganizationId),
		ServiceId:          config.GetString(config.ServiceId),
		GroupId:            f.org.GetGroupIdString(),
		CurrentBlockNumber: big.NewInt(100),
	}
	payment.AuthToken, _ = f.validator.NewToken(payment.Address, &payment.UserID, nil)
	var err error
	payment.AuthTokenParsed, payment.AuthTokenExpiryBlockNumber, err = ParseToken(payment.AuthToken)
	require.NoError(t, err)
	signPaymentRequest(payment, f.userKey)
	return payment
}

func signPaymentRequest(payment *Payment, key *ecdsa.PrivateKey) {
	message := bytes.Join([][]byte{
		[]byte(PrefixSignature), []byte(payment.Address), []byte(payment.UserID),
		[]byte(payment.OrganizationId), []byte(payment.ServiceId), []byte(payment.GroupId),
		bigIntToBytes(payment.CurrentBlockNumber), payment.AuthToken,
	}, nil)
	payment.Signature = getSignature(message, key)
}

func signTokenRequest(request *GetFreeCallTokenRequest, groupID string, key *ecdsa.PrivateKey) {
	message := bytes.Join([][]byte{
		[]byte(PrefixSignature), []byte(request.GetAddress()), []byte(request.GetUserId()),
		[]byte(config.GetString(config.OrganizationId)), []byte(config.GetString(config.ServiceId)),
		[]byte(groupID), bigIntToBytes(new(big.Int).SetUint64(request.CurrentBlock)),
	}, nil)
	request.Signature = getSignature(message, key)
}
