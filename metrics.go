// Package grpcprom provides Prometheus instrumentation for gRPC clients and servers.
//
// The following metrics are provided:
//
//  grpc_client_connections_open [gauge] Number of gRPC client connections open.
//  grpc_client_connections_total [counter] Total number of gRPC client connections opened.
//  grpc_client_requests_pending{grpc_type,grpc_service,grpc_method} [gauge] Number of gRPC client requests pending.
//  grpc_client_requests_total{grpc_type,grpc_service,grpc_method,grpc_code} [counter] Total number of gRPC client requests completed.
//  grpc_client_latency_seconds{grpc_type,grpc_service,grpc_method,grpc_code} [histogram] Latency of gRPC client requests.
//  grpc_client_recv_bytes{grpc_type,grpc_service,grpc_method,grpc_frame} [histogram] Bytes received in gRPC client responses.
//  grpc_client_sent_bytes{grpc_type,grpc_service,grpc_method,grpc_frame} [histogram] Bytes sent in gRPC client requests.
//
//  grpc_server_connections_open [gauge] Number of gRPC server connections open.
//  grpc_server_connections_total [counter] Total number of gRPC server connections opened.
//  grpc_server_requests_pending{grpc_type,grpc_service,grpc_method} [gauge] Number of gRPC server requests pending.
//  grpc_server_requests_total{grpc_type,grpc_service,grpc_method,grpc_code} [counter] Total number of gRPC server requests completed.
//  grpc_server_latency_seconds{grpc_type,grpc_service,grpc_method,grpc_code} [histogram] Latency of gRPC server requests.
//  grpc_server_recv_bytes{grpc_type,grpc_service,grpc_method,grpc_frame} [histogram] Bytes received in gRPC server requests.
//  grpc_server_sent_bytes{grpc_type,grpc_service,grpc_method,grpc_frame} [histogram] Bytes sent in gRPC server responses.
package grpcprom

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
)

// AllCodes is a slice of all gRPC codes.
var AllCodes = []codes.Code{
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

// ClientMetrics is a collection of gRPC client metrics.
type ClientMetrics struct {
	handler *handler
}

// NewClientMetrics returns new ClientMetrics with the given options.
func NewClientMetrics(options ...Option) *ClientMetrics {
	return &ClientMetrics{
		handler: newMetrics("client", options...),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// to the provided channel and returns once the last descriptor has been sent.
func (m *ClientMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.handler.describe(ch)
}

// Collect sends each collected metric via the provided channel
// and returns once the last metric has been sent.
func (m *ClientMetrics) Collect(ch chan<- prometheus.Metric) {
	m.handler.collect(ch)
}

// StatsHandler returns a gRPC stats handler.
func (m *ClientMetrics) StatsHandler() stats.Handler {
	return m.handler
}

// StreamInterceptor returns a gRPC client stream interceptor.
func (m *ClientMetrics) StreamInterceptor() grpc.StreamClientInterceptor {
	return m.handler.streamClientInterceptor
}

// UnaryInterceptor returns a gRPC client unary interceptor.
func (m *ClientMetrics) UnaryInterceptor() grpc.UnaryClientInterceptor {
	return m.handler.unaryClientInterceptor
}

// Init initializes the metrics for srv with the given codes.
func (m *ClientMetrics) Init(srv *grpc.Server, codes ...codes.Code) {
	for srvName, info := range srv.GetServiceInfo() {
		m.handler.init(srvName, info.Methods, codes)
	}
}

// ServerMetrics is a collection of gRPC server metrics.
type ServerMetrics struct {
	handler *handler
}

// NewServerMetrics returns new ServerMetrics with the given options.
func NewServerMetrics(options ...Option) *ServerMetrics {
	return &ServerMetrics{
		handler: newMetrics("server", options...),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// to the provided channel and returns once the last descriptor has been sent.
func (m *ServerMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.handler.describe(ch)
}

// Collect sends each collected metric via the provided channel
// and returns once the last metric has been sent.
func (m *ServerMetrics) Collect(ch chan<- prometheus.Metric) {
	m.handler.collect(ch)
}

// Init initializes the metrics for srv with the given codes.
func (m *ServerMetrics) Init(srv *grpc.Server, codes ...codes.Code) {
	for srvName, info := range srv.GetServiceInfo() {
		m.handler.init(srvName, info.Methods, codes)
	}
}

// StatsHandler returns a gRPC stats handler.
func (m *ServerMetrics) StatsHandler() stats.Handler {
	return m.handler
}

// StreamInterceptor returns a gRPC server stream interceptor.
func (m *ServerMetrics) StreamInterceptor() grpc.StreamServerInterceptor {
	return m.handler.streamServerInterceptor
}

// UnaryInterceptor returns a gRPC server unary interceptor.
func (m *ServerMetrics) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return m.handler.unaryServerInterceptor
}

var (
	errNoop  = errors.New("noop metric")
	noopDesc = prometheus.NewInvalidDesc(errNoop)
)

type observer interface {
	prometheus.Collector
	Init(lvs ...string)
	Observe(value float64, lvs ...string)
}

type noopObserver struct{}

func (noopObserver) Describe(chan<- *prometheus.Desc)     {}
func (noopObserver) Collect(chan<- prometheus.Metric)     {}
func (noopObserver) Init(lvs ...string)                   {}
func (noopObserver) Observe(value float64, lvs ...string) {}

type histogram struct {
	m *prometheus.HistogramVec
}

func (h *histogram) Collect(ch chan<- prometheus.Metric) { h.m.Collect(ch) }
func (h *histogram) Describe(ch chan<- *prometheus.Desc) { h.m.Describe(ch) }
func (h *histogram) Observe(v float64, lvs ...string)    { h.m.WithLabelValues(lvs...).Observe(v) }
func (h *histogram) Init(lvs ...string)                  { h.m.GetMetricWithLabelValues(lvs...) }

// counters is a histogram without the buckets... sum and count only.
type counters struct {
	sum *prometheus.CounterVec
	num *prometheus.CounterVec
}

func (m *counters) Collect(ch chan<- prometheus.Metric) {
	m.sum.Collect(ch)
	m.num.Collect(ch)
}

func (m *counters) Describe(ch chan<- *prometheus.Desc) {
	m.sum.Describe(ch)
	m.num.Describe(ch)
}

func (m *counters) Observe(v float64, lvs ...string) {
	// TODO: make atomic
	m.sum.WithLabelValues(lvs...).Add(v)
	m.num.WithLabelValues(lvs...).Inc()
}

func (m *counters) Init(lvs ...string) {
	m.sum.GetMetricWithLabelValues(lvs...)
	m.num.GetMetricWithLabelValues(lvs...)
}

type noopCounter struct{}

func (noopCounter) Desc() *prometheus.Desc           { return noopDesc }
func (noopCounter) Write(*dto.Metric) error          { return errNoop }
func (noopCounter) Describe(chan<- *prometheus.Desc) {}
func (noopCounter) Collect(chan<- prometheus.Metric) {}
func (noopCounter) Set(float64)                      {}
func (noopCounter) Inc()                             {}
func (noopCounter) Dec()                             {}
func (noopCounter) Add(float64)                      {}
func (noopCounter) Sub(float64)                      {}
func (noopCounter) SetToCurrentTime()                {}

type noopGauge struct{}

func (noopGauge) Desc() *prometheus.Desc           { return noopDesc }
func (noopGauge) Write(*dto.Metric) error          { return errNoop }
func (noopGauge) Describe(chan<- *prometheus.Desc) {}
func (noopGauge) Collect(chan<- prometheus.Metric) {}
func (noopGauge) Set(float64)                      {}
func (noopGauge) Inc()                             {}
func (noopGauge) Dec()                             {}
func (noopGauge) Add(float64)                      {}
func (noopGauge) Sub(float64)                      {}
func (noopGauge) SetToCurrentTime()                {}

type counterVec interface {
	prometheus.Collector
	GetMetricWithLabelValues(lvs ...string) (prometheus.Counter, error)
	WithLabelValues(lvs ...string) prometheus.Counter
}

type noopCounterVec struct {
	noopCounter
}

func (v noopCounterVec) GetMetricWithLabelValues(lvs ...string) (prometheus.Counter, error) {
	return v, nil
}

func (v noopCounterVec) WithLabelValues(lvs ...string) prometheus.Counter {
	return v
}

type gaugeVec interface {
	prometheus.Collector
	GetMetricWithLabelValues(lvs ...string) (prometheus.Gauge, error)
	WithLabelValues(lvs ...string) prometheus.Gauge
}

type noopGaugeVec struct {
	noopGauge
}

func (v noopGaugeVec) GetMetricWithLabelValues(lvs ...string) (prometheus.Gauge, error) {
	return v, nil
}

func (v noopGaugeVec) WithLabelValues(lvs ...string) prometheus.Gauge {
	return v
}
