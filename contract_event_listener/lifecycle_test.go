package contractlistener

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/lifecycle"
	"github.com/stretchr/testify/require"
)

type shutdownSubscriptionService struct{ started, stopped chan struct{} }

func (s *shutdownSubscriptionService) Logs(ctx context.Context, _ json.RawMessage) (*rpc.Subscription, error) {
	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}
	sub := notifier.CreateSubscription()
	go func() {
		close(s.started)
		_ = notifier.Notify(sub.ID, types.Log{Topics: []common.Hash{crypto.Keccak256Hash([]byte("OrganizationModified(bytes32)")), {}}})
		<-sub.Err()
		close(s.stopped)
	}()
	return sub, nil
}

type shutdownWSProcessor struct {
	*blockchain.MockProcessor
	client *ethclient.Client
}

func (p *shutdownWSProcessor) GetEthWSClient() *ethclient.Client { return p.client }

func TestOrganizationListenerCancellationUnsubscribes(t *testing.T) {
	previous := config.Vip().Get("registry_address_key")
	config.Vip().Set("registry_address_key", "0x0000000000000000000000000000000000000001")
	t.Cleanup(func() { config.Vip().Set("registry_address_key", previous) })
	service := &shutdownSubscriptionService{started: make(chan struct{}), stopped: make(chan struct{})}
	rpcServer := rpc.NewServer()
	t.Cleanup(rpcServer.Stop)
	require.NoError(t, rpcServer.RegisterName("eth", service))
	server := httptest.NewServer(rpcServer.WebsocketHandler([]string{"*"}))
	t.Cleanup(server.Close)
	client, err := rpc.DialContext(t.Context(), "ws"+strings.TrimPrefix(server.URL, "http"))
	require.NoError(t, err)
	t.Cleanup(client.Close)
	metadataSeen := make(chan struct{})
	meta := orgMetadataFromJSON(t, orgGroupSameEndpoints)
	previousRead := getOrganizationMetaDataContext
	getOrganizationMetaDataContext = func(context.Context) (*blockchain.OrganizationMetaData, error) { close(metadataSeen); return meta, nil }
	t.Cleanup(func() { getOrganizationMetaDataContext = previousRead })
	listener := &ContractEventListener{
		BlockchainProcessor:         &shutdownWSProcessor{MockProcessor: blockchain.NewMockProcessor(true), client: ethclient.NewClient(client)},
		CurrentOrganizationMetaData: meta,
	}
	workers := lifecycle.New()
	t.Cleanup(func() { workers.Cancel(); workers.Wait() })
	require.True(t, workers.Go(listener.ListenOrganizationMetadataChangingContext))
	select {
	case <-service.started:
	case <-time.After(3 * time.Second):
		t.Fatal("contract listener did not subscribe")
	}
	select {
	case <-metadataSeen:
	case <-time.After(time.Second):
		t.Fatal("contract listener did not process the subscribed event")
	}
	workers.Cancel()
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("contract listener did not stop")
	}
	select {
	case <-service.stopped:
	case <-time.After(time.Second):
		t.Fatal("contract listener did not unsubscribe")
	}
}
