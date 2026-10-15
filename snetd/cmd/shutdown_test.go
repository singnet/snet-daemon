package cmd

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const shutdownMethod = "/shutdown.Service/Call"

// Keep payment validation enabled; only Ethereum is replaced.
type shutdownProcessor struct {
	*blockchain.MockProcessor
}

func (p *shutdownProcessor) MultiPartyEscrowChannel(*big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
	return &blockchain.MultiPartyEscrowChannel{
		Sender:    common.HexToAddress("0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf"),
		Recipient: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		Value:     big.NewInt(100), Nonce: big.NewInt(0), Expiration: big.NewInt(1000),
	}, true, nil
}

type shutdownListener struct {
	net.Listener
	once    sync.Once
	closed  chan struct{}
	onClose func()
}

func (l *shutdownListener) Close() error {
	err := l.Listener.Close()
	l.once.Do(func() {
		close(l.closed)
		if l.onClose != nil {
			l.onClose()
		}
	})
	return err
}

func shutdownComponents(t *testing.T, backend string, store storage.AtomicStorage) *Components {
	t.Helper()
	conf := lifecycleConfig(t)
	conf.Set(config.BlockchainEnabledKey, true)
	conf.Set(config.PassthroughEnabledKey, true)
	conf.Set(config.ServiceEndpointKey, "http://"+backend)
	conf.Set(config.ServiceTimeout, "10s")
	components := lifecycleComponents(t)
	components.blockchain = &shutdownProcessor{blockchain.NewMockProcessor(true)}
	components.atomicStorage = store
	return components
}

func seedShutdownChannel(t *testing.T, components *Components) {
	t.Helper()
	channel, ok, err := components.PaymentChannelService().PaymentChannel(&escrow.PaymentChannelKey{ID: big.NewInt(42)})
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, escrow.NewPaymentChannelStorage(components.MPESpecificStorage()).Put(
		&escrow.PaymentChannelKey{ID: big.NewInt(42)}, channel))
}

func shutdownPaymentContext(t *testing.T, ctx context.Context, amount int64) context.Context {
	t.Helper()
	key, err := crypto.HexToECDSA(strings.Repeat("0", 63) + "1")
	require.NoError(t, err)
	message := bytes.Join([][]byte{
		[]byte(escrow.PrefixInSignature), common.Address{}.Bytes(),
		common.BigToHash(big.NewInt(42)).Bytes(), common.BigToHash(big.NewInt(0)).Bytes(),
		common.BigToHash(big.NewInt(amount)).Bytes(),
	}, nil)
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		handler.PaymentTypeHeader, escrow.EscrowPaymentType,
		handler.PaymentChannelIDHeader, "42", handler.PaymentChannelNonceHeader, "0",
		handler.PaymentChannelAmountHeader, fmt.Sprint(amount),
		handler.PaymentChannelSignatureHeader, string(utils.GetSignature(message, key)),
	))
}

func shutdownClient(t *testing.T, address string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func shutdownCall(ctx context.Context, conn *grpc.ClientConn) error {
	return conn.Invoke(ctx, shutdownMethod, &emptypb.Empty{}, &emptypb.Empty{})
}

func assertShutdownChannel(t *testing.T, components *Components, locked bool, amount int64) {
	t.Helper()
	key := &escrow.PaymentChannelKey{ID: big.NewInt(42)}
	lock, acquired, err := storage.NewLocker(components.LockerStorage()).Lock(key.String())
	require.NoError(t, err)
	if acquired {
		require.NoError(t, lock.Unlock())
	}
	require.Equal(t, !locked, acquired, "payment channel lock")
	channel, present, err := escrow.NewPaymentChannelStorage(components.MPESpecificStorage()).Get(key)
	require.NoError(t, err)
	require.True(t, present)
	require.Zero(t, channel.AuthorizedAmount.Cmp(big.NewInt(amount)), "authorized payment amount")
}

func waitShutdown(t *testing.T, done <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func shutdownAmount(t *testing.T, outcome string, err error) int64 {
	t.Helper()
	switch outcome {
	case "success":
		require.NoError(t, err)
		return 1
	case "error":
		require.Equal(t, codes.Internal, status.Code(err))
	case "cancel":
		require.Equal(t, codes.Canceled, status.Code(err))
	case "timeout":
		require.Error(t, err)
	default:
		t.Fatalf("unknown shutdown outcome %q", outcome)
	}
	return 0
}

func shutdownBackend(t *testing.T, outcome string) (string, <-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var enteredOnce, releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	lis := testListener(t)
	server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
		if err := stream.RecvMsg(&emptypb.Empty{}); err != nil {
			return err
		}
		enteredOnce.Do(func() { close(entered) })
		select {
		case <-release:
		case <-stream.Context().Done():
			return status.FromContextError(stream.Context().Err()).Err()
		}
		if outcome == "error" {
			return status.Error(codes.Internal, "controlled backend failure")
		}
		return stream.SendMsg(&emptypb.Empty{})
	}))
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(func() { unblock(); server.Stop() })
	return lis.Addr().String(), entered, unblock
}

func TestDaemonShutdownCompletesPayment(t *testing.T) {
	endpoint := shutdownEtcd(t)
	for _, setup := range []struct {
		name        string
		originalMux bool
		etcd        bool
	}{
		{name: "memory/fork_mux"},
		{name: "memory/original_mux", originalMux: true},
		{name: "etcd/fork_mux", etcd: true},
		{name: "etcd/original_mux", originalMux: true, etcd: true},
	} {
		for _, outcome := range []string{"success", "error", "cancel"} {
			t.Run(setup.name+"/"+outcome, func(t *testing.T) {
				backend, entered, unblock := shutdownBackend(t, outcome)
				var components *Components
				if setup.etcd {
					components = shutdownEtcdComponents(t, backend, endpoint, t.Name())
				} else {
					components = shutdownComponents(t, backend, storage.NewMemStorage())
				}
				config.Vip().Set(config.Experimental, map[string]any{"use_original_cmux": setup.originalMux})
				seedShutdownChannel(t, components)
				d, err := newDaemon(components)
				require.NoError(t, err)
				listener := &shutdownListener{Listener: d.lis, closed: make(chan struct{})}
				d.lis = listener
				d.start()
				t.Cleanup(func() { unblock(); d.grpcServer.Stop() })

				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				client := shutdownClient(t, d.lis.Addr().String())
				callDone := make(chan error, 1)
				paymentCtx := shutdownPaymentContext(t, ctx, 1)
				go func() { callDone <- shutdownCall(paymentCtx, client) }()
				waitShutdown(t, entered, "the paid backend call")
				assertShutdownChannel(t, components, true, 0)

				stopped := make(chan struct{})
				go func() { d.stop(); close(stopped) }()
				waitShutdown(t, listener.closed, "shutdown to close the listener")
				newCtx, newCancel := context.WithTimeout(t.Context(), time.Second)
				defer newCancel()
				require.Equal(t, codes.Unavailable, status.Code(shutdownCall(shutdownPaymentContext(t, newCtx, 1),
					shutdownClient(t, d.lis.Addr().String()))), "shutdown must reject new calls")
				select {
				case <-stopped:
					t.Fatal("shutdown returned while the paid call was still running")
				default:
				}
				assertShutdownChannel(t, components, true, 0)
				if outcome == "cancel" {
					cancel()
				} else {
					unblock()
				}
				waitShutdown(t, stopped, "payment cleanup and daemon shutdown")
				amount := shutdownAmount(t, outcome, <-callDone)
				assertShutdownChannel(t, components, false, amount)

				// A fresh daemon must be able to reuse the same channel and storage.
				config.Vip().Set(config.PassthroughEnabledKey, false)
				restarted, err := newDaemon(components)
				require.NoError(t, err)
				restarted.start()
				t.Cleanup(restarted.stop)
				retryCtx, retryCancel := context.WithTimeout(t.Context(), 3*time.Second)
				defer retryCancel()
				require.NoError(t, shutdownCall(shutdownPaymentContext(t, retryCtx, amount+1),
					shutdownClient(t, restarted.lis.Addr().String())))
				restarted.stop()
				assertShutdownChannel(t, components, false, amount+1)
			})
		}
	}
}
