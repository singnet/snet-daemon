package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/singnet/snet-daemon/v6/freecall"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/training"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

type lifecycleTransport func(*http.Request) (*http.Response, error)

func (f lifecycleTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Isolate global configuration and intercept version checks so tests stay offline.
func lifecycleConfig(t *testing.T) *viper.Viper {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	t.Chdir(repoRoot)
	previous, transport := config.Vip(), http.DefaultTransport
	conf := viper.New()
	require.NoError(t, conf.MergeConfigMap(previous.AllSettings()))
	config.SetVip(conf)
	http.DefaultTransport = lifecycleTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api.github.com" {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"tag_name":"test-version"}`)), Header: make(http.Header)}, nil
		}
		if r.URL.Hostname() == "127.0.0.1" || r.URL.Hostname() == "localhost" {
			return transport.RoundTrip(r)
		}
		return nil, fmt.Errorf("unexpected external request: %s", r.URL)
	})
	t.Cleanup(func() { config.SetVip(previous); http.DefaultTransport = transport })
	for key, value := range map[string]any{
		config.DaemonTypeKey: "grpc", config.DaemonEndpoint: "127.0.0.1:0",
		config.ServiceEndpointKey: "http://127.0.0.1:1", config.DaemonGroupName: "default_group",
		config.BlockchainEnabledKey: false, config.PaymentChannelStorageTypeKey: "memory",
		config.BlockChainNetworkSelected: "sepolia",
		config.AllowedUserFlag:           false, config.MeteringEnabled: false, config.EnableDynamicPricing: false,
		config.ModelTrainingEnabled: false, config.ServiceHeartbeatType: "none", config.HeartbeatServiceEndpoint: "",
		config.SSLCertPathKey: "", config.SSLKeyPathKey: "", config.AutoSSLDomainKey: "",
		config.NotificationServiceEndpoint: "", config.AlertsEMail: "", config.Experimental: map[string]any{},
		config.PvtKeyForFreeCalls: "", config.TokenSecretKey: strings.Repeat("s", 32),
		config.PaymentChannelStorageServerKey + ".enabled": false,
	} {
		conf.Set(key, value)
	}
	return conf
}

func lifecycleComponents(t *testing.T) *Components {
	t.Helper()
	service, err := blockchain.InitServiceMetaDataFromJson([]byte(`{
		"version":1,"encoding":"proto","service_type":"grpc","mpe_address":"0x0000000000000000000000000000000000000001",
		"groups":[{"group_name":"default_group","group_id":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"free_calls":2,"free_call_signer_address":"0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf",
		"pricing":[{"price_model":"fixed_price","price_in_cogs":1,"default":true}]}]
	}`))
	require.NoError(t, err)
	org, err := blockchain.InitOrganizationMetaDataFromJson([]byte(`{
		"org_name":"test","org_id":"test-org","groups":[{"group_name":"default_group",
		"group_id":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","payment":{
		"payment_address":"0x0000000000000000000000000000000000000001","payment_expiration_threshold":0,
		"payment_channel_storage_client":{"connection_timeout":"100ms","request_timeout":"100ms","endpoints":["http://127.0.0.1:1"]}}}]
	}`))
	require.NoError(t, err)
	components := &Components{serviceMetadata: service, organizationMetaData: org, blockchain: blockchain.NewMockProcessor(true)}
	t.Cleanup(components.Close)
	return components
}

func TestComponentsAssemblePaymentAndTrainingServices(t *testing.T) {
	conf := lifecycleConfig(t)
	conf.Set(config.BlockchainEnabledKey, true)
	conf.Set(config.PvtKeyForFreeCalls, strings.Repeat("0", 63)+"1")
	components := lifecycleComponents(t)
	for _, tc := range []struct {
		name, paymentType string
		get               func() handler.StreamPaymentHandler
	}{
		{"escrow", escrow.EscrowPaymentType, components.EscrowPaymentHandler},
		{"free call", freecall.PaymentType, components.FreeCallPaymentHandler},
		{"prepaid", escrow.PrePaidPaymentType, components.PrePaidPaymentHandler},
		{"training", escrow.TrainPaymentType, components.TrainStreamPaymentHandler},
		{"allowed users", escrow.EscrowPaymentType, components.AllowedUserPaymentHandler},
	} {
		t.Run(tc.name, func(t *testing.T) {
			created := tc.get()
			require.Equal(t, tc.paymentType, created.Type())
			require.Same(t, created, tc.get())
		})
	}
	unary := components.TrainUnaryPaymentHandler()
	require.Equal(t, escrow.TrainPaymentType, unary.Type())
	require.Same(t, unary, components.TrainUnaryPaymentHandler())
	require.Same(t, components.PaymentChannelService(), components.PaymentChannelService())
	require.Same(t, components.FreeCallUserService(), components.FreeCallUserService())
	users, err := components.FreeCallUserService().ListUsers()
	require.NoError(t, err)
	require.Empty(t, users)
	userKey, err := components.FreeCallUserService().UserKey(&freecall.Payment{UserID: "test-user", Address: "0x0000000000000000000000000000000000000001", OrganizationId: "test-org", ServiceId: "test-service"})
	require.NoError(t, err)
	require.Equal(t, components.OrganizationMetaData().GetGroupIdString(), userKey.GroupID)
	require.Equal(t, "test-org", userKey.OrganizationId)
	require.Same(t, components.PrePaidService(), components.PrePaidService())
	require.Same(t, components.PaymentChannelStateService(), components.PaymentChannelStateService())
	require.Same(t, components.ProviderControlService(), components.ProviderControlService())
	require.IsType(t, &freecall.StateService{}, components.FreeCallStateService())
	require.Same(t, components.FreeCallStateService(), components.FreeCallStateService())
	require.IsType(t, &escrow.TokenService{}, components.TokenService())
	require.Same(t, components.TokenService(), components.TokenService())
	require.IsType(t, &training.NoTrainingDaemonServer{}, components.TrainingService())
	require.Same(t, components.TrainingService(), components.TrainingService())
	trainingMetadata, err := components.DaemonHeartBeat().TrainingMetadata()
	require.ErrorContains(t, err, "service end point")
	require.False(t, trainingMetadata.TrainingEnabled)
	require.Same(t, components.ModelStorage(), components.ModelStorage())
	require.Same(t, components.ModelUserStorage(), components.ModelUserStorage())
	require.Same(t, components.PendingModelStorage(), components.PendingModelStorage())
	require.Same(t, components.PublicModelStorage(), components.PublicModelStorage())
	require.Same(t, components.ConfigurationService(), components.ConfigurationService())
	require.Same(t, components.PricingStrategy(), components.PricingStrategy())
	price, err := components.PricingStrategy().GetPrice(&handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "/example.Service/Ping"}})
	require.NoError(t, err)
	require.Zero(t, price.Cmp(big.NewInt(1)))
}

func TestNewDaemonRejectsInvalidRuntimeSettings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings map[string]any
		message  string
	}{
		{"heartbeat", map[string]any{config.ServiceHeartbeatType: "invalid"}, "heartbeat"},
		{"notifications", map[string]any{config.NotificationServiceEndpoint: "invalid", config.AlertsEMail: "test@example.com"}, "must be valid"},
		{"occupied address", nil, "Error binding"},
		{"missing TLS files", map[string]any{config.SSLCertPathKey: "missing-cert.pem", config.SSLKeyPathKey: "missing-key.pem"}, "unable to load"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := lifecycleConfig(t)
			for key, value := range tc.settings {
				conf.Set(key, value)
			}
			if tc.name == "occupied address" {
				listener := testListener(t)
				conf.Set(config.DaemonEndpoint, listener.Addr().String())
			}
			d, err := newDaemon(&Components{blockchain: blockchain.NewMockProcessor(false)})
			if d.lis != nil {
				t.Cleanup(func() { _ = d.lis.Close() })
			}
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestNewDaemonLoadsTLSCertificate(t *testing.T) {
	conf := lifecycleConfig(t)
	cert, key, cleanup := generateSelfSignedCert(t)
	defer cleanup()
	conf.Set(config.SSLCertPathKey, cert)
	conf.Set(config.SSLKeyPathKey, key)
	components := &Components{blockchain: blockchain.NewMockProcessor(false)}
	d, err := newDaemon(components)
	require.NoError(t, err)
	t.Cleanup(d.stop)
	require.NotEmpty(t, d.sslCert.Certificate)
	require.Same(t, components, d.components)
}

func TestNewDaemonTrafficSplitDefersListenerBinding(t *testing.T) {
	conf := lifecycleConfig(t)
	conf.Set(config.Experimental, map[string]any{"traffic_split": map[string]any{"grpc": 0, "http": 0}})
	d, err := newDaemon(&Components{blockchain: blockchain.NewMockProcessor(false)})
	require.NoError(t, err)
	require.Nil(t, d.lis)
	require.NotNil(t, d.blockProc)
}

func TestDaemonServesHealthWithPaymentServices(t *testing.T) {
	conf := lifecycleConfig(t)
	serviceListener := testListener(t)
	conf.Set(config.ServiceEndpointKey, "http://"+serviceListener.Addr().String())
	conf.Set(config.BlockchainEnabledKey, true)
	conf.Set(config.PvtKeyForFreeCalls, strings.Repeat("0", 63)+"1")
	components := lifecycleComponents(t)
	d, err := newDaemon(components)
	require.NoError(t, err)
	t.Cleanup(d.stop)
	d.start()
	conn, err := grpc.NewClient(d.lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close()) }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	reply, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, reply.Status)
	require.NotNil(t, components.grpcUnaryInterceptor)
	require.NotNil(t, components.grpcStreamInterceptor)
	called := false
	unary := components.GrpcUnaryInterceptor()
	_, err = unary(metadata.NewIncomingContext(context.Background(), metadata.MD{}), nil, &grpc.UnaryServerInfo{FullMethod: "/example.Service/Ping"}, func(context.Context, any) (any, error) { called = true; return nil, nil })
	require.NoError(t, err)
	require.True(t, called)
}

func TestInitComponentsPanicsForMissingConfiguration(t *testing.T) {
	lifecycleConfig(t)
	command := &cobra.Command{}
	command.Flags().String("config", filepath.Join(t.TempDir(), "missing.json"), "")
	require.NoError(t, command.Flags().Set("config", command.Flags().Lookup("config").Value.String()))
	require.Panics(t, func() { InitComponents(command) })
}

func TestComponentsBlockchainInitializesDisabledProcessor(t *testing.T) {
	lifecycleConfig(t)
	components := &Components{}
	processor := components.Blockchain()
	t.Cleanup(components.Close)
	require.False(t, processor.Enabled())
	require.Same(t, processor, components.Blockchain())
}

func TestComponentsEtcdServerRejectsInvalidConfiguration(t *testing.T) {
	for _, scenario := range []string{"parse error", "start error"} {
		t.Run(scenario, func(t *testing.T) {
			conf := lifecycleConfig(t)
			if scenario == "parse error" {
				conf.Set(config.PaymentChannelStorageServerKey+".client_port", "not-a-port")
			} else {
				conf.Set(config.PaymentChannelStorageServerKey+".enabled", true)
				conf.Set(config.PaymentChannelStorageServerKey+".client_port", -1)
				conf.Set(config.PaymentChannelStorageServerKey+".peer_port", -1)
				conf.Set(config.PaymentChannelStorageServerKey+".data_dir", t.TempDir())
				conf.Set(config.PaymentChannelStorageServerKey+".log_outputs", []string{"stderr"})
				conf.Set(config.PaymentChannelStorageServerKey+".log_level", "error")
			}
			components := &Components{}
			require.Panics(t, func() { components.EtcdServer() })
			require.Nil(t, components.etcdServer)
		})
	}
}

func TestComponentsMeteringAuthenticationAndCaching(t *testing.T) {
	for _, allowed := range []bool{true, false} {
		t.Run(fmt.Sprint(allowed), func(t *testing.T) {
			conf := lifecycleConfig(t)
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.Path != "/metering/verify" || r.Header.Get("X-Signature") == "" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if !allowed {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				_, _ = io.WriteString(w, `{"data":"success"}`)
			}))
			defer server.Close()
			conf.Set(config.BlockchainEnabledKey, true)
			conf.Set(config.MeteringEnabled, true)
			conf.Set(config.MeteringEndpoint, server.URL)
			conf.Set(config.PvtKeyForMetering, strings.Repeat("0", 63)+"1")
			components := lifecycleComponents(t)
			if allowed {
				require.NotNil(t, components.GrpcStreamInterceptor())
				require.NotNil(t, components.GrpcStreamInterceptor())
			} else {
				require.Panics(t, func() { components.GrpcStreamInterceptor() })
				require.Nil(t, components.grpcStreamInterceptor)
			}
			require.EqualValues(t, 1, requests.Load(), "authentication must run once for a cached interceptor")
		})
	}
}

type blockFailureProcessor struct {
	blockchain.Processor
	err error
}

func (p blockFailureProcessor) CurrentBlock() (*big.Int, error) { return nil, p.err }

func TestMeteringVerificationRejectsURLAndBlockErrors(t *testing.T) {
	conf := lifecycleConfig(t)
	conf.Set(config.PvtKeyForMetering, strings.Repeat("0", 63)+"1")
	components := &Components{blockchain: blockchain.NewMockProcessor(true)}
	allowed, err := components.verifyAuthenticationSetUpForFreeCall("http://%invalid", "group")
	require.False(t, allowed)
	require.Error(t, err)
	expected := errors.New("block unavailable")
	components.blockchain = blockFailureProcessor{Processor: blockchain.NewMockProcessor(true), err: expected}
	allowed, err = components.verifyAuthenticationSetUpForFreeCall("http://127.0.0.1:1", "group")
	require.False(t, allowed)
	require.ErrorIs(t, err, expected)
}

func TestComponentsFreeCallStateRequiresMatchingSigner(t *testing.T) {
	for _, key := range []string{"", strings.Repeat("0", 63) + "2"} {
		t.Run(fmt.Sprint(len(key)), func(t *testing.T) {
			conf := lifecycleConfig(t)
			conf.Set(config.BlockchainEnabledKey, true)
			conf.Set(config.PvtKeyForFreeCalls, key)
			components := lifecycleComponents(t)
			require.IsType(t, &freecall.DisabledStateService{}, components.FreeCallStateService())
			require.Nil(t, components.freeCallStateService)
		})
	}
}

func TestDaemonHTTPServesRequestsThroughTLS(t *testing.T) {
	conf := lifecycleConfig(t)
	conf.Set(config.DaemonTypeKey, "http")
	conf.Set(config.PassthroughEnabledKey, false)
	cert, key, cleanup := generateSelfSignedCert(t)
	defer cleanup()
	conf.Set(config.SSLCertPathKey, cert)
	conf.Set(config.SSLKeyPathKey, key)
	d, err := newDaemon(&Components{blockchain: blockchain.NewMockProcessor(false)})
	require.NoError(t, err)
	t.Cleanup(d.stop)
	d.start()
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} // Local, self-signed test certificate.
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	response, err := client.Post("https://"+d.lis.Addr().String(), "text/plain", strings.NewReader("TLS echo"))
	require.NoError(t, err)
	defer func() { require.NoError(t, response.Body.Close()) }()
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "TLS echo", string(data))
}

func TestComponentsLocalEtcdPersistsAtomicStorage(t *testing.T) {
	conf := lifecycleConfig(t)
	clientListener, peerListener := testListener(t), testListener(t)
	clientPort := clientListener.Addr().(*net.TCPAddr).Port
	peerPort := peerListener.Addr().(*net.TCPAddr).Port
	require.NoError(t, clientListener.Close())
	require.NoError(t, peerListener.Close())
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", clientPort)
	for key, value := range map[string]any{
		config.PaymentChannelStorageServerKey + ".enabled":         true,
		config.PaymentChannelStorageServerKey + ".client_port":     clientPort,
		config.PaymentChannelStorageServerKey + ".peer_port":       peerPort,
		config.PaymentChannelStorageServerKey + ".cluster":         fmt.Sprintf("storage-1=http://127.0.0.1:%d", peerPort),
		config.PaymentChannelStorageServerKey + ".data_dir":        t.TempDir(),
		config.PaymentChannelStorageServerKey + ".startup_timeout": "5s",
		config.PaymentChannelStorageServerKey + ".log_level":       "error",
		config.PaymentChannelStorageServerKey + ".log_outputs":     []string{"stderr"},
		config.PaymentChannelStorageClientKey + ".endpoints":       []string{endpoint},
		config.PaymentChannelStorageClientKey + ".request_timeout": "3s",
		config.PaymentChannelStorageTypeKey:                        "etcd",
	} {
		conf.Set(key, value)
	}
	components := lifecycleComponents(t)
	server := components.EtcdServer()
	require.NotNil(t, server)
	require.Same(t, server, components.EtcdServer())
	store := components.AtomicStorage()
	require.NoError(t, store.Put("test-key", "stored-value"))
	value, present, err := store.Get("test-key")
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, "stored-value", value)
}
