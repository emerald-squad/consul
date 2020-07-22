package consul

import (
	"net"

	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

func newGRPCServer(listener net.Listener) (interface{}, error) {
	lis := &grpcListener{
		addr:  s.Listener.Addr(),
		conns: make(chan net.Conn),
	}

	// We don't need to pass tls.Config to the server since it's multiplexed
	// behind the RPC listener, which already has TLS configured.
	srv := grpc.NewServer(
		grpc.StatsHandler(grpcStatsHandler),
		grpc.StreamInterceptor(GRPCCountingStreamInterceptor),
	)
	agentpb.RegisterConsulServer(srv, NewGRPCService(s))
	if s.config.GRPCTestServerEnabled {
		agentpb.RegisterTestServer(srv, &GRPCTest{srv: s})
	}

	go srv.Serve(lis)
	s.GRPCListener = lis

	// Set up a gRPC client connection to the above listener.
	dialer := newDialer(s.serverLookup, s.tlsConfigurator.OutgoingRPCWrapper())
	conn, err := grpc.Dial(lis.Addr().String(),
		grpc.WithInsecure(),
		grpc.WithContextDialer(dialer),
		grpc.WithDisableRetry(),
		grpc.WithStatsHandler(grpcStatsHandler),
		grpc.WithBalancerName("pick_first"))
	if err != nil {
		return err
	}

	s.grpcConn = conn

	return nil
}

// TODO: document how each of the methods is used.
type grpcListener struct {
	conns chan net.Conn
	addr  net.Addr
}

func (l *grpcListener) Handle(conn net.Conn) {
	l.conns <- conn
}

func (l *grpcListener) Accept() (net.Conn, error) {
	return <-l.conns, nil
}

func (l *grpcListener) Addr() net.Addr {
	return l.addr
}

func (l *grpcListener) Close() error {
	return nil
}

// TODO: grpcListener implementation for when grpc is not enabled.
type noopGRPCListener struct {
	logger hclog.Logger
}

func (l *noopGRPCListener) Handle(conn net.Conn) {
	l.logger.Error("GRPC conn opened but GRPC is not enabled, closing",
		"conn", logConn(conn),
	)
	conn.Close()
}
