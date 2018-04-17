

# grpcprom
`import "github.com/abursavich/grpcprom"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Examples](#pkg-examples)

## <a name="pkg-overview">Overview</a>
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



#### <a name="example_">Example</a>

Code:
``` go
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
```


## <a name="pkg-index">Index</a>
* [Variables](#pkg-variables)
* [type HistogramOpts](#HistogramOpts)
* [type Metrics](#Metrics)
  * [func NewMetrics(options MetricsOpts) *Metrics](#NewMetrics)
  * [func (m *Metrics) Collect(ch chan&lt;- prometheus.Metric)](#Metrics.Collect)
  * [func (m *Metrics) Describe(ch chan&lt;- *prometheus.Desc)](#Metrics.Describe)
  * [func (m *Metrics) DialOption() grpc.DialOption](#Metrics.DialOption)
  * [func (m *Metrics) InitServer(srv *grpc.Server, code ...codes.Code)](#Metrics.InitServer)
  * [func (m *Metrics) ServerOption() grpc.ServerOption](#Metrics.ServerOption)
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




## <a name="HistogramOpts">type</a> [HistogramOpts](/grpcprom.go?s=2582:2657#L69)
``` go
type HistogramOpts struct {
    Buckets []float64
    Disable bool
    // contains filtered or unexported fields
}
```
HistogramOpts specify options for histograms.










## <a name="Metrics">type</a> [Metrics](/grpcprom.go?s=3017:3057#L94)
``` go
type Metrics struct {
    // contains filtered or unexported fields
}
```
Metrics track gRPC metrics.







### <a name="NewMetrics">func</a> [NewMetrics](/grpcprom.go?s=3117:3162#L99)
``` go
func NewMetrics(options MetricsOpts) *Metrics
```
NewMetrics returns new metrics with the given options.





### <a name="Metrics.Collect">func</a> (\*Metrics) [Collect](/grpcprom.go?s=3802:3856#L119)
``` go
func (m *Metrics) Collect(ch chan<- prometheus.Metric)
```
Collect sends each collected metric via the provided channel
and returns once the last metric has been sent.

It implements the prometheus.Collector interface.




### <a name="Metrics.Describe">func</a> (\*Metrics) [Describe](/grpcprom.go?s=3509:3563#L110)
``` go
func (m *Metrics) Describe(ch chan<- *prometheus.Desc)
```
Describe sends the super-set of all possible descriptors of metrics
to the provided channel and returns once the last descriptor has been sent.

It implements the prometheus.Collector interface.




### <a name="Metrics.DialOption">func</a> (\*Metrics) [DialOption](/grpcprom.go?s=5378:5424#L165)
``` go
func (m *Metrics) DialOption() grpc.DialOption
```
DialOption returns a gRPC DialOption that instruments metrics
for the client connection.




### <a name="Metrics.InitServer">func</a> (\*Metrics) [InitServer](/grpcprom.go?s=4091:4157#L127)
``` go
func (m *Metrics) InitServer(srv *grpc.Server, code ...codes.Code)
```
InitServer initializes the metrics exported by the server.
It limits the code labels to those provided. If not provided,
all known code labels are initialized.




### <a name="Metrics.ServerOption">func</a> (\*Metrics) [ServerOption](/grpcprom.go?s=5560:5610#L171)
``` go
func (m *Metrics) ServerOption() grpc.ServerOption
```
ServerOption returns a gRPC ServerOption that instruments metrics
for the server.




### <a name="Metrics.StatsHandler">func</a> (\*Metrics) [StatsHandler](/grpcprom.go?s=5092:5138#L158)
``` go
func (m *Metrics) StatsHandler() stats.Handler
```
StatsHandler returns a gRPC stats.Handler.

Deprecated: Use DialOption or ServerOption instead.




## <a name="MetricsOpts">type</a> [MetricsOpts](/grpcprom.go?s=2900:2984#L86)
``` go
type MetricsOpts struct {
    Client SubsystemOpts
    Server SubsystemOpts
    // contains filtered or unexported fields
}
```
MetricsOpts specify options for metrics.










## <a name="SubsystemOpts">type</a> [SubsystemOpts](/grpcprom.go?s=2737:2854#L77)
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
