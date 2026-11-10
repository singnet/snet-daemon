package etcddb

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/server/v3/embed"
)

func TestEtcdClusterQuorum(t *testing.T) {
	servers, endpoints := startEtcdTestCluster(t)
	metadata := newEtcdTestMetadata(t, endpoints, 3*time.Second)
	client, err := NewEtcdClientFromVip(nil, metadata)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	assertValue := func(key, expected string) {
		t.Helper()
		value, present, err := client.Get(key)
		require.NoError(t, err)
		require.True(t, present)
		require.Equal(t, expected, value)
	}

	t.Log("write and read with all three members running")
	require.NoError(t, client.Put("before-failure", "three members"))
	assertValue("before-failure", "three members")

	// Stop the leader so the two remaining members must elect a replacement.
	leaderIndex := -1
	require.Eventually(t, func() bool {
		for i, server := range servers {
			if server.etcd.Server.Leader() == server.etcd.Server.MemberID() {
				leaderIndex = i
				return true
			}
		}
		return false
	}, 10*time.Second, 50*time.Millisecond, "the cluster must elect a leader")
	oldLeader := servers[leaderIndex].etcd.Server.MemberID()
	servers[leaderIndex].Close()
	servers[leaderIndex] = nil
	require.Eventually(t, func() bool {
		for _, server := range servers {
			if server == nil {
				continue
			}
			leader := server.etcd.Server.Leader()
			if leader == 0 || leader == oldLeader {
				return false
			}
		}
		return true
	}, 10*time.Second, 50*time.Millisecond, "the remaining members must elect a new leader")

	t.Log("existing data and new writes remain available with two members")
	assertValue("before-failure", "three members")
	require.NoError(t, client.Put("after-first-failure", "two members"))
	assertValue("after-first-failure", "two members")

	for i, server := range servers {
		if server != nil {
			server.Close()
			servers[i] = nil
			break
		}
	}
	for i, server := range servers {
		if server == nil {
			continue
		}
		// Address only the live member, proving that failures below are due
		// to loss of quorum rather than connecting to a stopped endpoint.
		client.etcd.SetEndpoints(endpoints[i])
		ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
		status, err := client.etcd.Status(ctx, endpoints[i])
		cancel()
		require.NoError(t, err, "the remaining member must still be reachable")
		require.Equal(t, uint64(server.etcd.Server.MemberID()), status.Header.MemberId)
	}

	t.Log("writes and linearizable reads fail with only one member")
	require.Error(t, client.Put("after-second-failure", "no quorum"))
	// Read a previously committed key: a successful stale read is not enough.
	value, present, err := client.Get("after-first-failure")
	require.Error(t, err)
	require.False(t, present)
	require.Empty(t, value)
}

func newEtcdTestMetadata(t *testing.T, endpoints []string, timeout time.Duration) *blockchain.OrganizationMetaData {
	t.Helper()
	metadataJSON, err := json.Marshal(blockchain.OrganizationMetaData{
		OrgName: "quorum-test",
		OrgID:   "quorum-test",
		Groups: []blockchain.Group{{
			GroupName: config.GetString(config.DaemonGroupName),
			GroupID:   "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
			PaymentDetails: blockchain.Payment{
				PaymentAddress:            "0x671276c61943A35D5F230d076bDFd91B0c47bF09",
				PaymentChannelStorageType: "etcd",
				PaymentChannelStorageClient: blockchain.PaymentChannelStorageClient{
					ConnectionTimeout: "5s",
					RequestTimeout:    timeout.String(),
					Endpoints:         endpoints,
				},
			},
		}},
	})
	require.NoError(t, err)
	metadata, err := blockchain.InitOrganizationMetaDataFromJson(metadataJSON)
	require.NoError(t, err)
	return metadata
}

func startEtcdTestCluster(t *testing.T) ([]*EtcdServer, []string) {
	t.Helper()
	servers, endpoints := newEtcdTestCluster(t, 3)
	// Start all members before waiting for readiness, which needs a quorum.
	// Retain each embedded server immediately so cleanup also covers startup failures.
	for i, server := range servers {
		var err error
		server.etcd, err = embed.StartEtcd(getEtcdConf(server.conf))
		require.NoError(t, err, "start member %d", i)
	}
	deadline := time.NewTimer(servers[0].conf.StartupTimeout)
	defer deadline.Stop()
	for i, server := range servers {
		select {
		case <-server.etcd.Server.ReadyNotify():
		case err := <-server.etcd.Err():
			t.Fatalf("member %d failed before becoming ready: %v", i, err)
		case <-deadline.C:
			t.Fatalf("member %d took too long to become ready", i)
		}
	}
	return servers, endpoints
}

func newEtcdTestCluster(t *testing.T, size int) ([]*EtcdServer, []string) {
	t.Helper()
	dir := t.TempDir()
	servers := make([]*EtcdServer, size)
	endpoints := make([]string, size)
	peers := make([]string, size)
	var listeners []net.Listener
	reservePort := func() int {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = listener.Close() })
		listeners = append(listeners, listener)
		return listener.Addr().(*net.TCPAddr).Port
	}
	for i := range size {
		conf := &EtcdServerConf{
			ID:             fmt.Sprintf("quorum-member-%d", i),
			Scheme:         "http",
			Host:           "127.0.0.1",
			ClientPort:     reservePort(),
			PeerPort:       reservePort(),
			Token:          dir,
			StartupTimeout: 15 * time.Second,
			DataDir:        filepath.Join(dir, fmt.Sprintf("member-%d", i)),
			LogLevel:       "error",
			LogOutputs:     []string{"stderr"},
		}
		servers[i] = &EtcdServer{conf: conf}
		endpoints[i] = fmt.Sprintf("http://%s:%d", conf.Host, conf.ClientPort)
		peers[i] = fmt.Sprintf("%s=http://%s:%d", conf.ID, conf.Host, conf.PeerPort)
	}
	t.Cleanup(func() {
		for _, server := range servers {
			if server != nil && server.etcd != nil {
				server.Close()
			}
		}
	})
	for _, server := range servers {
		server.conf.Cluster = strings.Join(peers, ",")
	}
	// Hold all ports until the configuration is complete so they are distinct.
	for _, listener := range listeners {
		require.NoError(t, listener.Close())
	}
	return servers, endpoints
}
