package configuration_service

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/require"
)

func TestConfigurationServiceControlOperationsBroadcastAuthenticatedChanges(t *testing.T) {
	originalBlockchainEnabled := config.GetBool(config.BlockchainEnabledKey)
	config.Vip().Set(config.BlockchainEnabledKey, false)
	t.Cleanup(func() { config.Vip().Set(config.BlockchainEnabledKey, originalBlockchainEnabled) })

	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	broadcaster := NewChannelBroadcaster()
	subscriber := broadcaster.NewSubscriber()
	service := ConfigurationService{
		authenticationAddressList: []common.Address{crypto.PubkeyToAddress(privateKey.PublicKey)},
		broadcast:                 broadcaster,
	}
	authentication := func(prefix string) *CallerAuthentication {
		const blockNumber = uint64(10)
		return &CallerAuthentication{
			CurrentBlock: blockNumber,
			Signature:    utils.GetSignature(service.getMessageBytes(prefix, blockNumber), privateKey),
		}
	}

	stopped, err := service.StopProcessingRequests(context.Background(), &EmptyRequest{Auth: authentication("_StopProcessingRequests")})
	require.NoError(t, err)
	require.Equal(t, StatusResponse_HAS_STOPPED_PROCESSING_REQUESTS, stopped.CurrentProcessingStatus)
	select {
	case notification := <-subscriber:
		require.Equal(t, StopProcessingAnyRequest, notification)
	case <-time.After(time.Second):
		t.Fatal("configuration stop notification was not broadcast")
	}

	started, err := service.StartProcessingRequests(context.Background(), &EmptyRequest{Auth: authentication("_StartProcessingRequests")})
	require.NoError(t, err)
	require.Equal(t, StatusResponse_REQUEST_IN_PROGRESS, started.CurrentProcessingStatus)
	select {
	case notification := <-subscriber:
		require.Equal(t, StartProcessingAnyRequest, notification)
	case <-time.After(time.Second):
		t.Fatal("configuration start notification was not broadcast")
	}

	configuration, err := service.GetConfiguration(context.Background(), &EmptyRequest{Auth: authentication("_GetConfiguration")})
	require.NoError(t, err)
	require.NotEmpty(t, configuration.GetSchema().GetDetails())
	require.NotEmpty(t, configuration.GetCurrentConfiguration())

	_, err = service.UpdateConfiguration(context.Background(), &UpdateRequest{Auth: authentication("_UpdateConfiguration")})
	require.EqualError(t, err, "work in progress")
	_, err = service.IsDaemonProcessingRequests(context.Background(), &EmptyRequest{Auth: authentication("_IsDaemonProcessingRequests")})
	require.EqualError(t, err, "work in progress")
}

func TestGetAuthenticationAddressSkipsInvalidConfigurationValues(t *testing.T) {
	originalAddresses := config.Vip().GetStringSlice(config.AuthenticationAddresses)
	config.Vip().Set(config.AuthenticationAddresses, []string{"invalid", "0x00000000000000000000000000000000000000ab"})
	t.Cleanup(func() { config.Vip().Set(config.AuthenticationAddresses, originalAddresses) })

	addresses := getAuthenticationAddress()

	require.Equal(t, []common.Address{common.HexToAddress("0x00000000000000000000000000000000000000ab")}, addresses)
}
