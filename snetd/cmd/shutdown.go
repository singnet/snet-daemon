package cmd

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/lifecycle"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type daemonConnections struct {
	mu     sync.Mutex
	conns  map[*daemonConnection]struct{}
	closed bool
}
type daemonConnection struct {
	net.Conn
	owner *daemonConnections
	once  sync.Once
}
type daemonListener struct {
	net.Listener
	owner *daemonConnections
}

func (l *daemonListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	tracked := &daemonConnection{Conn: c, owner: l.owner}
	l.owner.mu.Lock()
	if l.owner.closed {
		l.owner.mu.Unlock()
		_ = c.Close()
		return nil, net.ErrClosed
	}
	if l.owner.conns == nil {
		l.owner.conns = make(map[*daemonConnection]struct{})
	}
	l.owner.conns[tracked] = struct{}{}
	l.owner.mu.Unlock()
	return tracked, nil
}

func (c *daemonConnection) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.owner.mu.Lock(); delete(c.owner.conns, c); c.owner.mu.Unlock() })
	return err
}

func (pool *daemonConnections) close() {
	pool.mu.Lock()
	pool.closed = true
	connections := make([]*daemonConnection, 0, len(pool.conns))
	for c := range pool.conns {
		connections = append(connections, c)
	}
	pool.mu.Unlock()
	for _, c := range connections {
		_ = c.Close()
	}
}

func (d *daemon) ownListener(l net.Listener) net.Listener {
	tracked := &daemonListener{Listener: l, owner: &d.connections}
	d.listeners = append(d.listeners, tracked)
	return tracked
}

func (d *daemon) runServer(serve func() error) {
	d.servers.Add(1)
	go func() {
		defer d.servers.Done()
		if err := serve(); err != nil && !d.stopping.Load() {
			zap.L().Error("daemon server stopped unexpectedly", zap.Error(err))
		}
	}()
}

func (d *daemon) serveHTTP(l net.Listener, handler http.Handler) {
	server := &http.Server{Handler: d.trackHTTP(handler), ReadHeaderTimeout: 10 * time.Second}
	d.httpServers = append(d.httpServers, server)
	d.runServer(func() error { return server.Serve(l) })
}

func (d *daemon) trackHTTP(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, done, ok := d.requests.Begin(r.Context())
		if !ok {
			http.Error(w, "daemon is shutting down", http.StatusServiceUnavailable)
			return
		}
		defer done()
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

type lifecycleStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *lifecycleStream) Context() context.Context { return s.ctx }

func (d *daemon) trackStream(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, next grpc.StreamHandler) error {
	ctx, done, ok := d.requests.Begin(stream.Context())
	if !ok {
		return status.Error(codes.Unavailable, "daemon is shutting down")
	}
	defer done()
	return next(srv, &lifecycleStream{ServerStream: stream, ctx: ctx})
}

func (d *daemon) trackUnary(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
	ctx, done, ok := d.requests.Begin(ctx)
	if !ok {
		return nil, status.Error(codes.Unavailable, "daemon is shutting down")
	}
	defer done()
	return next(ctx, req)
}

func (d *daemon) stop() {
	d.stopOnce.Do(func() {
		d.stopping.Store(true)
		if d.requests == nil {
			d.requests = lifecycle.New()
		}
		d.requests.StopAccepting()
		if d.components != nil {
			d.components.Background().Cancel()
		}
		for _, l := range d.listeners {
			_ = l.Close()
		}
		if d.lis != nil {
			_ = d.lis.Close()
		}
		if d.acmeListener != nil {
			_ = d.acmeListener.Close()
		}

		ctx, cancel := context.WithTimeout(context.Background(), config.ShutdownGracePeriod())
		defer cancel()
		drained := make(chan struct{})
		go func() {
			var httpDrains sync.WaitGroup
			for _, server := range d.httpServers {
				httpDrains.Add(1)
				go func() { defer httpDrains.Done(); _ = server.Shutdown(ctx) }()
			}
			httpDrains.Wait()
			// ServeHTTP transports must drain before GracefulStop touches the shared server.
			if d.grpcServer != nil {
				d.grpcServer.GracefulStop()
			}
			d.requests.Wait()
			close(drained)
		}()
		select {
		case <-drained:
		case <-ctx.Done():
			zap.L().Warn("graceful shutdown deadline reached; cancelling active requests")
			d.requests.Cancel()
			if d.components != nil {
				d.components.Calls().Cancel()
			}
			for _, server := range d.httpServers {
				_ = server.Close()
			}
			if d.grpcServer != nil {
				d.grpcServer.Stop()
			}
			d.connections.close()
			// Storage stays open until all payment defers have completed.
			<-drained
		}
		d.requests.Cancel()
		if d.closeProxy != nil {
			d.closeProxy()
		}
		d.connections.close()
		d.servers.Wait()
		if d.components != nil {
			d.components.Calls().Cancel()
			d.components.Background().Wait()
			if d.components.configurationBroadcaster != nil {
				d.components.configurationBroadcaster.Close()
			}
		}
	})
}
