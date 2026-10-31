package contractlistener

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/stretchr/testify/require"
)

type reconnectSubscriptionService struct {
	connections chan *rpc.Notifier
	reject      bool
}

func (s *reconnectSubscriptionService) Logs(ctx context.Context, _ json.RawMessage) (*rpc.Subscription, error) {
	if s.reject {
		return nil, errors.New("subscription rejected")
	}
	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}
	sub := notifier.CreateSubscription()
	go func() {
		// Separate events ensure metadata read failures do not terminate the subscription.
		for range 2 {
			_ = notifier.Notify(sub.ID, types.Log{Topics: []common.Hash{
				crypto.Keccak256Hash([]byte("OrganizationModified(bytes32)")), {},
			}})
		}
		s.connections <- notifier
	}()
	return sub, nil
}

type reconnectProcessor struct {
	*blockchain.MockProcessor
	client  *ethclient.Client
	connect func(context.Context) error
}

func (p *reconnectProcessor) GetEthWSClient() *ethclient.Client { return p.client }
func (p *reconnectProcessor) ConnectToWsClientContext(ctx context.Context) error {
	return p.connect(ctx)
}

func reconnectTestServer(t *testing.T, reject bool) (string, *reconnectSubscriptionService) {
	t.Helper()
	previous := config.Vip().Get("registry_address_key")
	config.Vip().Set("registry_address_key", "0x0000000000000000000000000000000000000001")
	t.Cleanup(func() { config.Vip().Set("registry_address_key", previous) })
	service := &reconnectSubscriptionService{connections: make(chan *rpc.Notifier, 4), reject: reject}
	server := rpc.NewServer()
	t.Cleanup(server.Stop)
	require.NoError(t, server.RegisterName("eth", service))
	httpServer := httptest.NewServer(server.WebsocketHandler([]string{"*"}))
	t.Cleanup(httpServer.Close)
	return "ws" + strings.TrimPrefix(httpServer.URL, "http"), service
}

func dialListenerClient(t *testing.T, endpoint string) *ethclient.Client {
	t.Helper()
	client, err := ethclient.DialContext(t.Context(), endpoint)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}

func TestOrganizationListenerReconnectsAfterDisconnect(t *testing.T) {
	endpoint, service := reconnectTestServer(t, false)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	processor := &reconnectProcessor{MockProcessor: blockchain.NewMockProcessor(true), client: dialListenerClient(t, endpoint)}
	initialClient := processor.client
	reconnected := make(chan struct{}, 1)
	processor.connect = func(ctx context.Context) error {
		client, err := ethclient.DialContext(ctx, endpoint)
		if err == nil {
			processor.client = client
			reconnected <- struct{}{}
		}
		return err
	}
	meta := orgMetadataFromJSON(t, orgGroupSameEndpoints)
	previousRead := getOrganizationMetaDataContext
	getOrganizationMetaDataContext = func(context.Context) (*blockchain.OrganizationMetaData, error) { return meta, nil }
	t.Cleanup(func() { getOrganizationMetaDataContext = previousRead })
	listener := &ContractEventListener{BlockchainProcessor: processor, CurrentOrganizationMetaData: meta}
	done := make(chan struct{})
	go func() { defer close(done); listener.ListenOrganizationMetadataChangingContext(ctx) }()
	t.Cleanup(func() { cancel(); <-done; processor.client.Close() })
	select {
	case <-service.connections:
		initialClient.Close()
	case <-ctx.Done():
		t.Fatal("initial subscription did not start")
	}
	select {
	case <-reconnected:
	case <-ctx.Done():
		t.Fatal("listener did not reconnect after disconnection")
	}
	select {
	case <-service.connections:
	case <-ctx.Done():
		t.Fatal("listener did not subscribe on the new connection")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("listener did not stop after reconnecting")
	}
}

func TestOrganizationListenerContinuesAfterMetadataReadError(t *testing.T) {
	endpoint, _ := reconnectTestServer(t, false)
	client := dialListenerClient(t, endpoint)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	meta := orgMetadataFromJSON(t, orgGroupSameEndpoints)
	reads := 0
	previousRead := getOrganizationMetaDataContext
	getOrganizationMetaDataContext = func(context.Context) (*blockchain.OrganizationMetaData, error) {
		reads++
		if reads == 1 {
			return nil, errors.New("IPFS unavailable")
		}
		cancel()
		return meta, nil
	}
	t.Cleanup(func() { getOrganizationMetaDataContext = previousRead })
	listener := &ContractEventListener{
		BlockchainProcessor:         &shutdownWSProcessor{MockProcessor: blockchain.NewMockProcessor(true), client: client},
		CurrentOrganizationMetaData: meta,
	}
	listener.watchOrganization(ctx)
	require.Equal(t, 2, reads)
	require.Same(t, meta, listener.CurrentOrganizationMetaData)
}

func TestOrganizationListenerConnectionFailures(t *testing.T) {
	for _, scenario := range []string{"connect failure", "nil client", "reconnect failure", "subscription failure", "already canceled"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
			defer cancel()
			processor := &reconnectProcessor{MockProcessor: blockchain.NewMockProcessor(true)}
			calls := 0
			processor.connect = func(context.Context) error {
				calls++
				if scenario == "nil client" {
					return nil
				}
				if calls == 2 {
					cancel()
				}
				return errors.New("websocket unavailable")
			}
			if scenario == "subscription failure" {
				endpoint, _ := reconnectTestServer(t, true)
				processor.client = dialListenerClient(t, endpoint)
			}
			if scenario == "already canceled" {
				cancel()
			}
			listener := &ContractEventListener{BlockchainProcessor: processor, CurrentOrganizationMetaData: orgMetadataFromJSON(t, orgGroupSameEndpoints)}
			if scenario == "reconnect failure" {
				listener.ListenOrganizationMetadataChangingContext(ctx)
				require.Equal(t, 2, calls)
			} else {
				listener.watchOrganization(ctx)
				want := 1
				if scenario == "already canceled" || scenario == "subscription failure" {
					want = 0
				}
				require.Equal(t, want, calls)
			}
		})
	}
}
