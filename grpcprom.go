// Package grpcprom provides Prometheus instrumentation for gRPC clients and servers.
//
// The following metrics are provided:
//
//  grpc_client_connections_open [gauge] Number of gRPC client connections open.
//  grpc_client_connections_total [counter] Total number of gRPC client connections opened.
//  grpc_client_requests_pending{service,method} [gauge] Number of gRPC client requests pending.
//  grpc_client_requests_total{service,method,code} [counter] Total number of gRPC client requests completed.
//  grpc_client_latency_seconds{service,method,code} [histogram] Latency of gRPC client requests.
//  grpc_client_recv_bytes{service,method,frame} [histogram] Bytes received in gRPC client responses.
//  grpc_client_sent_bytes{service,method,frame} [histogram] Bytes sent in gRPC client requests.
//
//  grpc_server_connections_open [gauge] Number of gRPC server connections open.
//  grpc_server_connections_total [counter] Total number of gRPC server connections opened.
//  grpc_server_requests_pending{service,method} [gauge] Number of gRPC server requests pending.
//  grpc_server_requests_total{service,method,code} [counter] Total number of gRPC server requests completed.
//  grpc_server_latency_seconds{service,method,code} [histogram] Latency of gRPC server requests.
//  grpc_server_recv_bytes{service,method,frame} [histogram] Bytes received in gRPC server requests.
//  grpc_server_sent_bytes{service,method,frame} [histogram] Bytes sent in gRPC server responses.
package grpcprom

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

const (
	header  = "header"
	payload = "payload"
	trailer = "trailer"
)

// DefaultLatencyBuckets are the default latency histogram buckets.
var DefaultLatencyBuckets = []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// DefaultBytesBuckets are the default bytes histogram buckets.
var DefaultBytesBuckets = []float64{0, 32, 64, 128, 256, 512, 1024, 2048, 8192, 32768, 131072, 524288}

var allCodes = []codes.Code{
	codes.OK,
	codes.Canceled,
	codes.Unknown,
	codes.InvalidArgument,
	codes.DeadlineExceeded,
	codes.NotFound,
	codes.AlreadyExists,
	codes.PermissionDenied,
	codes.ResourceExhausted,
	codes.FailedPrecondition,
	codes.Aborted,
	codes.OutOfRange,
	codes.Unimplemented,
	codes.Internal,
	codes.Unavailable,
	codes.DataLoss,
	codes.Unauthenticated,
}

// HistogramOpts specify options for histograms.
type HistogramOpts struct {
	_ struct{}

	Buckets []float64
	Disable bool
}

// SubsystemOpts specify options for gRPC subsystems (e.g. client or server).
type SubsystemOpts struct {
	_ struct{}

	BytesRecv HistogramOpts
	BytesSent HistogramOpts
	Latency   HistogramOpts
}

// MetricsOpts specify options for metrics.
type MetricsOpts struct {
	_ struct{}

	Client SubsystemOpts
	Server SubsystemOpts
}

// Metrics track gRPC metrics.
type Metrics struct {
	handler handler
}

// NewMetrics returns new metrics with the given options.
func NewMetrics(options MetricsOpts) *Metrics {
	return &Metrics{handler: handler{
		client: newMetrics("client", options.Client),
		server: newMetrics("server", options.Server),
	}}
}

// Describe sends the super-set of all possible descriptors of metrics
// to the provided channel and returns once the last descriptor has been sent.
//
// It implements the prometheus.Collector interface.
func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.handler.client.Describe(ch)
	m.handler.server.Describe(ch)
}

// Collect sends each collected metric via the provided channel
// and returns once the last metric has been sent.
//
// It implements the prometheus.Collector interface.
func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.handler.client.Collect(ch)
	m.handler.server.Collect(ch)
}

// InitServer initializes the metrics exported by the server.
// It limits the code labels to those provided. If not provided,
// all known code labels are initialized.
func (m *Metrics) InitServer(srv *grpc.Server, code ...codes.Code) {
	if code == nil {
		code = allCodes
	}
	frames := [...]string{header, payload, trailer}
	for srvName, info := range srv.GetServiceInfo() {
		for _, meth := range info.Methods {
			m.handler.server.reqsPending.GetMetricWithLabelValues(srvName, meth.Name)
			for _, c := range code {
				m.handler.server.reqsTotal.GetMetricWithLabelValues(srvName, meth.Name, c.String())
			}
			if m.handler.server.latency != nil {
				m.handler.server.latency.GetMetricWithLabelValues(srvName, meth.Name)
			}
			if m.handler.server.latency != nil {
				for _, f := range frames {
					m.handler.server.bytesRecv.GetMetricWithLabelValues(srvName, meth.Name, f)
				}
			}
			if m.handler.server.latency != nil {
				for _, f := range frames {
					m.handler.server.bytesSent.GetMetricWithLabelValues(srvName, meth.Name, f)
				}
			}
		}
	}
}

// StatsHandler returns a gRPC stats.Handler.
//
// Deprecated: Use DialOption or ServerOption instead.
func (m *Metrics) StatsHandler() stats.Handler {
	log.Print("grpcprom: WARNING: StatsHandler is deprecated and may be removed. Use DialOption or ServerOption instead.")
	return &m.handler
}

// DialOption returns a gRPC DialOption that instruments metrics
// for the client connection.
func (m *Metrics) DialOption() grpc.DialOption {
	return grpc.WithStatsHandler(&m.handler)
}

// ServerOption returns a gRPC ServerOption that instruments metrics
// for the server.
func (m *Metrics) ServerOption() grpc.ServerOption {
	return grpc.StatsHandler(&m.handler)
}

type metrics struct {
	connsOpen   prometheus.Gauge
	connsTotal  prometheus.Counter
	reqsPending *prometheus.GaugeVec
	reqsTotal   *prometheus.CounterVec
	latency     *prometheus.HistogramVec
	bytesSent   *prometheus.HistogramVec
	bytesRecv   *prometheus.HistogramVec
}

func newMetrics(subsys string, opts SubsystemOpts) *metrics {
	m := &metrics{
		connsOpen: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "connections_open",
				Help:      fmt.Sprintf("Number of gRPC %s connections open.", subsys),
			},
		),
		connsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "connections_total",
				Help:      fmt.Sprintf("Total number of gRPC %s connections opened.", subsys),
			},
		),
		reqsPending: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "requests_pending",
				Help:      fmt.Sprintf("Number of gRPC %s requests pending.", subsys),
			},
			[]string{"service", "method"},
		),
		reqsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "requests_total",
				Help:      fmt.Sprintf("Total number of gRPC %s requests completed.", subsys),
			},
			[]string{"service", "method", "code"},
		),
	}
	if !opts.Latency.Disable {
		if opts.Latency.Buckets == nil {
			opts.Latency.Buckets = DefaultLatencyBuckets
		}
		m.latency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "latency_seconds",
				Help:      fmt.Sprintf("Latency of gRPC %s requests.", subsys),
				Buckets:   opts.Latency.Buckets,
			},
			[]string{"service", "method", "code"},
		)
	}
	if !opts.BytesRecv.Disable {
		typ := "requests"
		if subsys == "client" {
			typ = "responses"
		}
		if opts.BytesRecv.Buckets == nil {
			opts.BytesRecv.Buckets = DefaultBytesBuckets
		}
		m.bytesRecv = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "recv_bytes",
				Help:      fmt.Sprintf("Bytes received in gRPC %s %s.", subsys, typ),
				Buckets:   opts.BytesRecv.Buckets,
			},
			[]string{"service", "method", "frame"},
		)
	}
	if !opts.BytesSent.Disable {
		typ := "responses"
		if subsys == "client" {
			typ = "requests"
		}
		if opts.BytesSent.Buckets == nil {
			opts.BytesSent.Buckets = DefaultBytesBuckets
		}
		m.bytesSent = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "sent_bytes",
				Help:      fmt.Sprintf("Bytes sent in gRPC %s %s.", subsys, typ),
				Buckets:   opts.BytesSent.Buckets,
			},
			[]string{"service", "method", "frame"},
		)
	}
	return m
}

func (m *metrics) Describe(ch chan<- *prometheus.Desc) {
	m.connsOpen.Describe(ch)
	m.connsTotal.Describe(ch)
	m.reqsPending.Describe(ch)
	m.reqsTotal.Describe(ch)
	if m.latency != nil {
		m.latency.Describe(ch)
	}
	if m.bytesSent != nil {
		m.bytesSent.Describe(ch)
	}
	if m.bytesRecv != nil {
		m.bytesRecv.Describe(ch)
	}
}

func (m *metrics) Collect(ch chan<- prometheus.Metric) {
	m.connsOpen.Collect(ch)
	m.connsTotal.Collect(ch)
	m.reqsPending.Collect(ch)
	m.reqsTotal.Collect(ch)
	if m.latency != nil {
		m.latency.Collect(ch)
	}
	if m.bytesSent != nil {
		m.bytesSent.Collect(ch)
	}
	if m.bytesRecv != nil {
		m.bytesRecv.Collect(ch)
	}
}

var rpcInfoKey = "rpc-tag"

type rpcInfo struct {
	server string
	method string
	begin  time.Time
}

// handler implements the stats.Handler interface.
type handler struct {
	client *metrics
	server *metrics
}

// TagRPC implements the stats.Handler interface.
func (*handler) TagRPC(ctx context.Context, v *stats.RPCTagInfo) context.Context {
	server, method := splitFullMethodName(v.FullMethodName)
	return context.WithValue(ctx, &rpcInfoKey, &rpcInfo{
		server: server,
		method: method,
	})
}

func splitFullMethodName(s string) (server, method string) {
	s = strings.TrimPrefix(s, "/")
	i := strings.Index(s, "/")
	if i < 0 {
		return "unknown", "unknown"
	}
	return s[:i], s[i+1:]
}

// HandleRPC implements the stats.Handler interface.
func (h *handler) HandleRPC(ctx context.Context, stat stats.RPCStats) {
	v, ok := ctx.Value(&rpcInfoKey).(*rpcInfo)
	if !ok {
		return
	}
	m := h.server
	if stat.IsClient() {
		m = h.client
	}
	switch s := stat.(type) {
	case *stats.Begin:
		v.begin = s.BeginTime
		m.reqsPending.WithLabelValues(v.server, v.method).Inc()
	case *stats.End:
		code := status.Code(s.Error).String()
		if m.latency != nil {
			m.latency.WithLabelValues(v.server, v.method, code).Observe(time.Since(v.begin).Seconds())
		}
		m.reqsTotal.WithLabelValues(v.server, v.method, code).Inc()
		m.reqsPending.WithLabelValues(v.server, v.method).Dec()
	case *stats.InHeader:
		if m.bytesRecv != nil {
			m.bytesRecv.WithLabelValues(v.server, v.method, header).Observe(float64(s.WireLength))
		}
	case *stats.InPayload:
		if m.bytesRecv != nil {
			m.bytesRecv.WithLabelValues(v.server, v.method, payload).Observe(float64(s.WireLength))
		}
	case *stats.InTrailer:
		if m.bytesRecv != nil {
			m.bytesRecv.WithLabelValues(v.server, v.method, trailer).Observe(float64(s.WireLength))
		}
	case *stats.OutHeader:
		if m.bytesSent != nil {
			m.bytesSent.WithLabelValues(v.server, v.method, header).Observe(0) // TODO ???
		}
	case *stats.OutPayload:
		if m.bytesSent != nil {
			m.bytesSent.WithLabelValues(v.server, v.method, payload).Observe(float64(s.WireLength))
		}
	case *stats.OutTrailer:
		if m.bytesSent != nil {
			m.bytesSent.WithLabelValues(v.server, v.method, trailer).Observe(float64(s.WireLength))
		}
	}
}

// TagConn implements the stats.Handler interface.
func (h *handler) TagConn(ctx context.Context, v *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn implements the stats.Handler interface.
func (h *handler) HandleConn(ctx context.Context, stat stats.ConnStats) {
	m := h.server
	if stat.IsClient() {
		m = h.client
	}
	switch stat.(type) {
	case *stats.ConnBegin:
		m.connsOpen.Inc()
		m.connsTotal.Inc()
	case *stats.ConnEnd:
		m.connsOpen.Dec()
	}
}
