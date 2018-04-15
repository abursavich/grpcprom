

# grpcprom
`import "github.com/abursavich/grpcprom"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Examples](#pkg-examples)

## <a name="pkg-overview">Overview</a>
Package grpcprom provides Prometheus instrumentation for gRPC servers.

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


#### <a name="example_">Example</a>
``` go
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
```



## <a name="pkg-index">Index</a>
* [Variables](#pkg-variables)
* [type HistogramOpts](#HistogramOpts)
* [type Metrics](#Metrics)
  * [func NewMetrics(options MetricsOpts) *Metrics](#NewMetrics)
  * [func (m *Metrics) Collect(ch chan&lt;- prometheus.Metric)](#Metrics.Collect)
  * [func (m *Metrics) Describe(ch chan&lt;- *prometheus.Desc)](#Metrics.Describe)
  * [func (m *Metrics) InitServer(srv *grpc.Server, code ...codes.Code)](#Metrics.InitServer)
  * [func (m *Metrics) StatsHandler() stats.Handler](#Metrics.StatsHandler)
* [type MetricsOpts](#MetricsOpts)
* [type SubsystemOpts](#SubsystemOpts)

#### <a name="pkg-examples">Examples</a>
* [Package](#example_)

#### <a name="pkg-files">Package files</a>
[grpcprom.go](/grpcprom.go) 



## <a name="pkg-variables">Variables</a>
``` go
var DefaultBytesBuckets = []float64{0, 32, 64, 128, 256, 512, 1024, 2048, 8192, 32768, 131072, 524288}
```
DefaultBytesBuckets are the default bytes histogram buckets.

``` go
var DefaultLatencyBuckets = []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
```
DefaultLatencyBuckets are the default latency histogram buckets.




## <a name="HistogramOpts">type</a> [HistogramOpts](/grpcprom.go?s=2563:2638#L68)
``` go
type HistogramOpts struct {
    Buckets []float64
    Disable bool
    // contains filtered or unexported fields
}
```
HistogramOpts specify options for histograms.










## <a name="Metrics">type</a> [Metrics](/grpcprom.go?s=2998:3038#L93)
``` go
type Metrics struct {
    // contains filtered or unexported fields
}
```
Metrics track gRPC metrics.







### <a name="NewMetrics">func</a> [NewMetrics](/grpcprom.go?s=3098:3143#L98)
``` go
func NewMetrics(options MetricsOpts) *Metrics
```
NewMetrics returns new metrics with the given options.





### <a name="Metrics.Collect">func</a> (\*Metrics) [Collect](/grpcprom.go?s=3783:3837#L118)
``` go
func (m *Metrics) Collect(ch chan<- prometheus.Metric)
```
Collect sends each collected metric via the provided channel
and returns once the last metric has been sent.

It implements the prometheus.Collector interface.




### <a name="Metrics.Describe">func</a> (\*Metrics) [Describe](/grpcprom.go?s=3490:3544#L109)
``` go
func (m *Metrics) Describe(ch chan<- *prometheus.Desc)
```
Describe sends the super-set of all possible descriptors of metrics
to the provided channel and returns once the last descriptor has been sent.

It implements the prometheus.Collector interface.




### <a name="Metrics.InitServer">func</a> (\*Metrics) [InitServer](/grpcprom.go?s=4072:4138#L126)
``` go
func (m *Metrics) InitServer(srv *grpc.Server, code ...codes.Code)
```
InitServer initializes the metrics exported by the server.
It limits the code labels to those provided. If not provided,
all known code labels are initialized.




### <a name="Metrics.StatsHandler">func</a> (\*Metrics) [StatsHandler](/grpcprom.go?s=5015:5061#L155)
``` go
func (m *Metrics) StatsHandler() stats.Handler
```
StatsHandler returns a gRPC stats.Handler.




## <a name="MetricsOpts">type</a> [MetricsOpts](/grpcprom.go?s=2881:2965#L85)
``` go
type MetricsOpts struct {
    Client SubsystemOpts
    Server SubsystemOpts
    // contains filtered or unexported fields
}
```
MetricsOpts specify options for metrics.










## <a name="SubsystemOpts">type</a> [SubsystemOpts](/grpcprom.go?s=2718:2835#L76)
``` go
type SubsystemOpts struct {
    BytesRecv HistogramOpts
    BytesSent HistogramOpts
    Latency   HistogramOpts
    // contains filtered or unexported fields
}
```
SubsystemOpts specify options for gRPC subsystems (e.g. client or server).














- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
