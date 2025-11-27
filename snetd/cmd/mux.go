package cmd

import (
	"net"

	"github.com/semyon-dev/cmux"
	originalCmux "github.com/soheilhy/cmux"
)

type ListenerType int

const (
	L_GRPC ListenerType = iota
	L_HTTP
	L_GRPC_WEB
)

type Endpoint struct {
	Type ListenerType
	L    net.Listener
}

type GRPCMux interface {
	Endpoints() []Endpoint
	Serve() error
}

// ---------------------------
// forkMux: cmux implementation using the forked version
// ---------------------------

type forkMux struct {
	mux      cmux.CMux
	splitWeb bool
}

func newForkMux(l net.Listener, splitWeb bool) GRPCMux {
	m := cmux.New(l)
	return &forkMux{mux: m, splitWeb: splitWeb}
}

func (m *forkMux) Endpoints() []Endpoint {
	eps := make([]Endpoint, 0, 3)

	// Match native gRPC (HTTP/2)
	grpcL := m.mux.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"),
	)
	eps = append(eps, Endpoint{Type: L_GRPC, L: grpcL})

	if m.splitWeb {
		// Match gRPC-Web (HTTP/1.1). Must be matched *before* generic HTTP1.
		grpcWebL := m.mux.Match(
			cmux.HTTP1HeaderFieldPrefix("content-type", "application/grpc-web"),
			cmux.HTTP1HeaderFieldPrefix("content-type", "application/grpc-web-text"),
		)
		eps = append(eps, Endpoint{Type: L_GRPC_WEB, L: grpcWebL})
	}

	// Match generic HTTP/1.1 (REST, heartbeat, encoding, OPTIONS, etc.)
	httpL := m.mux.Match(cmux.HTTP1Fast())
	eps = append(eps, Endpoint{Type: L_HTTP, L: httpL})

	return eps
}

func (m *forkMux) Serve() error { return m.mux.Serve() }

// ---------------------------
// origMux: cmux implementation using upstream cmux
// ---------------------------

type origMux struct {
	mux      originalCmux.CMux
	splitWeb bool
}

func newOriginalMux(l net.Listener, splitWeb bool) GRPCMux {
	m := originalCmux.New(l)
	return &origMux{mux: m, splitWeb: splitWeb}
}

func (m *origMux) Endpoints() []Endpoint {
	eps := make([]Endpoint, 0, 3)

	// Match native gRPC (HTTP/2)
	grpcL := m.mux.MatchWithWriters(
		originalCmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"),
	)
	eps = append(eps, Endpoint{Type: L_GRPC, L: grpcL})

	if m.splitWeb {
		// Match gRPC-Web (HTTP/1.1). Must be before generic HTTP1.
		grpcWebL := m.mux.Match(
			originalCmux.HTTP1HeaderFieldPrefix("content-type", "application/grpc-web"),
			originalCmux.HTTP1HeaderFieldPrefix("content-type", "application/grpc-web-text"),
		)
		eps = append(eps, Endpoint{Type: L_GRPC_WEB, L: grpcWebL})
	}

	// Match generic HTTP/1.1 traffic
	httpL := m.mux.Match(originalCmux.HTTP1Fast())
	eps = append(eps, Endpoint{Type: L_HTTP, L: httpL})

	return eps
}

func (m *origMux) Serve() error { return m.mux.Serve() }
