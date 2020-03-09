package grpcprom_test

import (
	"net"
	"net/http"

	"github.com/abursavich/grpcprom"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	bepb "github.com/abursavich/grpcprom/testdata/backend"
	fepb "github.com/abursavich/grpcprom/testdata/frontend"
)

func Example() {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewBuildInfoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	// Create gRPC client metrics and register with Prometheus.
	clientMetrics := grpcprom.NewClientMetrics()
	registry.MustRegister(clientMetrics)
	// Instrument gRPC client(s).
	backendConn, err := grpc.Dial(backendAddr,
		grpc.WithStatsHandler(clientMetrics.StatsHandler()),
		grpc.WithStreamInterceptor(clientMetrics.StreamInterceptor()),
		grpc.WithUnaryInterceptor(clientMetrics.UnaryInterceptor()),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		),
	)
	check(err)

	// Create gRPC server metrics and register with Prometheus.
	serverMetrics := grpcprom.NewServerMetrics()
	registry.MustRegister(serverMetrics)
	// Instrument gRPC server and initialize metrics.
	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(serverMetrics.StatsHandler()),
		grpc.StreamInterceptor(serverMetrics.StreamInterceptor()),
		grpc.UnaryInterceptor(serverMetrics.UnaryInterceptor()),
	)
	fepb.RegisterFrontendServer(grpcSrv, &FrontendServer{
		BackendClient: bepb.NewBackendClient(backendConn),
	})
	serverMetrics.Init(grpcSrv, codes.OK)

	// Serve metrics.
	httpLis, err := net.Listen("tcp", httpAddr)
	check(err)
	httpSrv := http.NewServeMux()
	httpSrv.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	go http.Serve(httpLis, httpSrv)

	// Serve gRPC.
	grpcLis, err := net.Listen("tcp", grpcAddr)
	check(err)
	check(grpcSrv.Serve(grpcLis))
}
