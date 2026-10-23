package cmd

import (
	"context"
	"encoding/base64"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/singnet/snet-daemon/v6/freecall"
	"github.com/singnet/snet-daemon/v6/training"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestComponentsVerifyMeteringConfigurations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/verify":
			assert.Equal(t, "verification", request.Header.Get("x-authtype"))
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"data":"success"}`))
		case "/failedresponse":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"data":"failed"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	originalEndpoint := config.GetString(config.MeteringEndpoint)
	originalKey := config.GetString(config.PvtKeyForMetering)
	config.Vip().Set(config.MeteringEndpoint, server.URL)
	t.Cleanup(func() {
		config.Vip().Set(config.MeteringEndpoint, originalEndpoint)
		config.Vip().Set(config.PvtKeyForMetering, originalKey)
	})

	component := &Components{blockchain: blockchain.NewMockProcessor(true)}
	ok, err := component.verifyAuthenticationSetUpForFreeCall(server.URL+"/verify", "testgroup")
	assert.EqualError(t, err, "you need a specify a valid private key 'pvt_key_for_metering' as part of service publication process.invalid length, need 256 bits")
	assert.False(t, ok)

	config.Vip().Set(config.PvtKeyForMetering, "6996606c7854992c10d8cdc9a13d511a9d9db8ab8f21e59d6ac901a76367b36b")
	ok, err = component.verifyAuthenticationSetUpForFreeCall(server.URL+"/verify", "testgroup")
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = component.verifyAuthenticationSetUpForFreeCall(server.URL+"/badurl", "")
	assert.EqualError(t, err, "Service call failed with status code : 404 ")
	assert.False(t, ok)

	ok, err = component.verifyAuthenticationSetUpForFreeCall(server.URL+"/failedresponse", "")
	assert.EqualError(t, err, "error returned by by Metering Service "+server.URL+"/verify Verification, pls check the pvt_key_for_metering set up. The public key in metering does not correspond to the private key in Daemon config")
	assert.False(t, ok)
}

func TestComponentsVerifyMeteringPrivateKeyFormats(t *testing.T) {
	oldConfig := config.Vip()
	config.SetVip(viper.New())
	t.Cleanup(func() { config.SetVip(oldConfig) })
	config.Vip().Set(config.OrganizationId, "test-org")
	config.Vip().Set(config.ServiceId, "test-service")

	requests := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Clone(context.Background())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"success"}`))
	}))
	defer server.Close()
	component := &Components{blockchain: blockchain.NewMockProcessor(true)}

	for _, prefix := range []string{"", "0x", "0X"} {
		t.Run("prefix="+prefix, func(t *testing.T) {
			config.Vip().Set(config.PvtKeyForMetering, prefix+strings.Repeat("0", 63)+"1")
			ok, err := component.verifyAuthenticationSetUpForFreeCall(server.URL, "test-group")
			require.NoError(t, err)
			require.True(t, ok)
			require.Len(t, requests, 1)
			request := <-requests
			require.Equal(t, "verification", request.Header.Get("x-authtype"))
			require.Equal(t, "test-org", request.Header.Get("X-Organizationid"))
			require.Equal(t, "test-service", request.Header.Get("X-Serviceid"))
			require.Equal(t, "test-group", request.Header.Get("X-Groupid"))
			signature, err := base64.StdEncoding.DecodeString(request.Header.Get("X-Signature"))
			require.NoError(t, err)
			block, ok := new(big.Int).SetString(request.Header.Get("X-Currentblocknumber"), 10)
			require.True(t, ok)
			message := append([]byte("_usage"+request.Header.Get("X-username")+"test-orgtest-servicetest-group"), block.FillBytes(make([]byte, 32))...)
			signer, err := utils.GetSignerAddressFromMessage(message, signature)
			require.NoError(t, err)
			require.Equal(t, common.HexToAddress("0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf"), *signer)
		})
	}

	for _, invalid := range []string{"", "0x", "0xinvalid", "0x" + strings.Repeat("0", 64)} {
		config.Vip().Set(config.PvtKeyForMetering, invalid)
		ok, err := component.verifyAuthenticationSetUpForFreeCall(server.URL, "test-group")
		require.Error(t, err)
		require.False(t, ok)
		require.Empty(t, requests, "an invalid key must be rejected before making an HTTP request")
	}
}

func TestComponentsBlockchainDisabledServices(t *testing.T) {
	originalBlockchainEnabled := config.GetBool(config.BlockchainEnabledKey)
	config.Vip().Set(config.BlockchainEnabledKey, false)
	t.Cleanup(func() {
		config.Vip().Set(config.BlockchainEnabledKey, originalBlockchainEnabled)
	})

	components := &Components{blockchain: blockchain.NewMockProcessor(false)}

	assert.IsType(t, &escrow.BlockChainDisabledStateService{}, components.PaymentChannelStateService())
	assert.IsType(t, &escrow.BlockChainDisabledProviderControlService{}, components.ProviderControlService())
	assert.IsType(t, &freecall.DisabledStateService{}, components.FreeCallStateService())
	assert.IsType(t, &escrow.BlockChainDisabledTokenService{}, components.TokenService())
	assert.IsType(t, &training.NoTrainingDaemonServer{}, components.TrainingService())
}

func TestComponentsPaymentValidationWhenBlockchainDisabled(t *testing.T) {
	originalAllowedUsers := config.GetBool(config.AllowedUserFlag)
	config.Vip().Set(config.AllowedUserFlag, false)
	t.Cleanup(func() {
		config.Vip().Set(config.AllowedUserFlag, originalAllowedUsers)
	})

	components := &Components{blockchain: blockchain.NewMockProcessor(false)}

	streamCalled := false
	streamInterceptor := components.GrpcStreamPaymentValidationInterceptor()
	err := streamInterceptor(nil, nil, &grpc.StreamServerInfo{}, func(any, grpc.ServerStream) error {
		streamCalled = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, streamCalled)

	unaryCalled := false
	unaryInterceptor := components.GrpcUnaryPaymentValidationInterceptor()
	_, err = unaryInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		unaryCalled = true
		return nil, nil
	})
	assert.NoError(t, err)
	assert.True(t, unaryCalled)
}
