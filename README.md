# grpcprom
[![License](https://img.shields.io/badge/license-mit-blue.svg?style=for-the-badge)](https://raw.githubusercontent.com/abursavich/grpcprom/main/LICENSE)
[![GoDev Reference](https://img.shields.io/static/v1?logo=go&logoColor=white&color=00ADD8&label=dev&message=reference&style=for-the-badge)](https://pkg.go.dev/bursavich.dev/grpcprom)
[![Go Report Card](https://goreportcard.com/badge/bursavich.dev/grpcprom?style=for-the-badge)](https://goreportcard.com/report/bursavich.dev/grpcprom)

Package grpcprom provides Prometheus instrumentation for gRPC clients and servers.

The following metrics are provided:


	grpc_client_connections_open [gauge] Number of gRPC client connections open.
	grpc_client_connections_total [counter] Total number of gRPC client connections opened.
	grpc_client_requests_pending{service,method} [gauge] Number of gRPC client requests pending.
	grpc_client_requests_total{service,method,code} [counter] Total number of gRPC client requests completed.
	grpc_client_latency_seconds{service,method,code} [histogram] Latency of gRPC client requests.
	grpc_client_recv_bytes{service,method,frame} [histogram] Bytes received in gRPC client responses.
	grpc_client_sent_bytes{service,method,frame} [histogram] Bytes sent in gRPC client requests.
	
	grpc_server_connections_open [gauge] Number of gRPC server connections open.
	grpc_server_connections_total [counter] Total number of gRPC server connections opened.
	grpc_server_requests_pending{service,method} [gauge] Number of gRPC server requests pending.
	grpc_server_requests_total{service,method,code} [counter] Total number of gRPC server requests completed.
	grpc_server_latency_seconds{service,method,code} [histogram] Latency of gRPC server requests.
	grpc_server_recv_bytes{service,method,frame} [histogram] Bytes received in gRPC server requests.
	grpc_server_sent_bytes{service,method,frame} [histogram] Bytes sent in gRPC server responses.


## Example

```go
registry := prometheus.NewRegistry()
registry.MustRegister(collectors.NewGoCollector())
registry.MustRegister(collectors.NewBuildInfoCollector())
registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

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
```
