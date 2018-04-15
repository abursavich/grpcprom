package grpcprom_test

import (
	"log"
	"net"

	"github.com/abursavich/grpcprom"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	bpb "github.com/abursavich/grpcprom/testdata/backend"
	pb "github.com/abursavich/grpcprom/testdata/frontend"
)

func Example() {
	// Create gRPC metrics with selected options and register with Prometheus.
	m := grpcprom.NewMetrics(grpcprom.MetricsOpts{
		// ...
	})
	prometheus.MustRegister(m)
	// Instrument gRPC client(s).
	conn, err := grpc.Dial(backendAddr, grpc.WithStatsHandler(m.StatsHandler()))
	if err != nil {
		log.Fatal(err)
	}
	// Instrument gRPC server and, optionally, initialize server metrics.
	srv := grpc.NewServer(grpc.StatsHandler(m.StatsHandler()))
	pb.RegisterFrontendServer(srv, &Server{
		backend: bpb.NewBackendClient(conn),
	})
	m.InitServer(srv)
	// Listen and serve.
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(srv.Serve(lis))
}
