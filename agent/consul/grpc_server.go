package consul

import (
	"net"

	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

func newGRPCHandler(addr net.Addr) *grpcHandler {
	conns := make(chan net.Conn)

	lis := &grpcListener{
		addr:  addr,
		conns: conns,
	}

	// We don't need to pass tls.Config to the server since it's multiplexed
	// behind the RPC listener, which already has TLS configured.
	srv := grpc.NewServer(
	// TODO(streaming): grpc.StatsHandler(grpcStatsHandler),
	// TODO(streaming): grpc.StreamInterceptor(GRPCCountingStreamInterceptor),
	)

	// TODO(streaming): add gRPC services to srv here

	handler := &grpcHandler{
		conns: conns,
		run: func() error {
			return srv.Serve(lis)
		},
		shutdown: func() error {
			srv.Stop()
			return nil
		},
	}
	return handler
}

// grpcHandler implements a handler for the rpc server listener, and the
// agent.Component interface for managing the lifecycle of the grpc.Server.
type grpcHandler struct {
	conns    chan net.Conn
	run      func() error
	shutdown func() error
}

// Handle the conenction by sending it to a channel for the grpc.Server to receive.
func (h *grpcHandler) Handle(conn net.Conn) {
	h.conns <- conn
}

func (h *grpcHandler) Run() error {
	return h.run()
}

func (h *grpcHandler) Shutdown() error {
	return h.shutdown()
}

// grpcListener implements net.Listener for grpc.Server.
type grpcListener struct {
	conns chan net.Conn
	addr  net.Addr
}

// Accept blocks until a connection is received from Handle, and then returns the
// connection. Accept implements part of the net.Listener interface for grpc.Server.
func (l *grpcListener) Accept() (net.Conn, error) {
	return <-l.conns, nil
}

func (l *grpcListener) Addr() net.Addr {
	return l.addr
}

// Close does nothing. The connections are managed by the caller.
func (l *grpcListener) Close() error {
	return nil
}

type noopGRPCHandler struct {
	logger hclog.Logger
}

func (h *noopGRPCHandler) Handle(conn net.Conn) {
	h.logger.Error("gRPC conn opened but gRPC RPC is disabled, closing",
		"conn", logConn(conn))
	_ = conn.Close()
}

func (h *noopGRPCHandler) Run() error {
	return nil
}

func (h *noopGRPCHandler) Shutdown() error {
	return nil
}
