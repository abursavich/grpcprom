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
	grpcMetrics := grpcprom.NewMetrics(grpcprom.MetricsOpts{
		// ...
	})
	prometheus.MustRegister(grpcMetrics)
	// Instrument gRPC client(s).
	backendConn, err := grpc.Dial(backendAddr, grpcMetrics.DialOption())
	if err != nil {
		log.Fatal(err)
	}
	// Instrument gRPC server and, optionally, initialize server metrics.
	srv := grpc.NewServer(grpcMetrics.ServerOption())
	pb.RegisterFrontendServer(srv, &Server{
		backend: bpb.NewBackendClient(backendConn),
	})
	grpcMetrics.InitServer(srv)
	// Listen and serve.
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(srv.Serve(lis))
}
