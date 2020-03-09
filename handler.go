package grpcprom

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

const (
	header  = "Header"
	payload = "Payload"
	trailer = "Trailer"
)

const (
	unknown      = "Unknown"
	unary        = "Unary"
	clientStream = "ClientStream"
	serverStream = "ServerStream"
	bidiStream   = "BidiStream"
)

type collectors []prometheus.Collector

func (c collectors) Describe(ch chan<- *prometheus.Desc) {
	for _, v := range c {
		v.Describe(ch)
	}
}

func (c collectors) Collect(ch chan<- prometheus.Metric) {
	for _, v := range c {
		v.Collect(ch)
	}
}

type handler struct {
	methods sync.Map // name => info

	connsOpen   prometheus.Gauge
	connsTotal  prometheus.Counter
	reqsPending *prometheus.GaugeVec
	reqsTotal   *prometheus.CounterVec
	latency     observer
	sentBytes   observer
	recvBytes   observer
}

func newMetrics(subsys string, opts ...Option) *handler {
	o := &options{
		latency: histogramOptions{
			buckets: DefaultLatencyBuckets,
		},
	}
	for _, opt := range opts {
		opt.applyOption(o)
	}
	return &handler{
		connsOpen:   newConnsOpen(subsys, o.connsOpen),
		connsTotal:  newConnsTotal(subsys, o.connsTotal),
		reqsPending: newReqsPending(subsys, o.reqsPending),
		reqsTotal:   newReqsTotal(subsys, o.reqsTotal),
		latency:     newLatency(subsys, o.latency),
		sentBytes:   newSentBytes(subsys, o.sentBytes),
		recvBytes:   newRecvBytes(subsys, o.recvBytes),
	}
}

func newConnsOpen(subsys string, opts metricOptions) prometheus.Gauge {
	if opts.disable {
		return nil
	}
	return prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "grpc",
			Subsystem: subsys,
			Name:      "connections_open",
			Help:      fmt.Sprintf("Number of gRPC %s connections open.", subsys),
		},
	)
}

func newConnsTotal(subsys string, opts metricOptions) prometheus.Gauge {
	if opts.disable {
		return nil
	}
	return prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "grpc",
			Subsystem: subsys,
			Name:      "connections_total",
			Help:      fmt.Sprintf("Total number of gRPC %s connections opened.", subsys),
		},
	)
}

func newReqsPending(subsys string, opts metricOptions) *prometheus.GaugeVec {
	if opts.disable {
		return nil
	}
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "grpc",
			Subsystem: subsys,
			Name:      "requests_pending",
			Help:      fmt.Sprintf("Number of gRPC %s requests pending.", subsys),
		},
		[]string{"grpc_type", "grpc_service", "grpc_method"},
	)
}

func newReqsTotal(subsys string, opts metricOptions) *prometheus.CounterVec {
	if opts.disable {
		return nil
	}
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "grpc",
			Subsystem: subsys,
			Name:      "requests_total",
			Help:      fmt.Sprintf("Total number of gRPC %s requests completed.", subsys),
		},
		[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_code"},
	)
}

func newLatency(subsys string, opts histogramOptions) observer {
	if opts.disable {
		return nil
	}
	if len(opts.buckets) > 0 {
		return &histogram{prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "latency_seconds",
				Help:      fmt.Sprintf("Latency of gRPC %s requests.", subsys),
				Buckets:   opts.buckets,
			},
			[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_code"},
		)}
	}
	return &counters{
		sum: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "latency_seconds_sum",
				Help:      fmt.Sprintf("Latency of gRPC %s requests sum.", subsys),
			},
			[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_code"},
		),
		num: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "latency_seconds_count",
				Help:      fmt.Sprintf("Latency of gRPC %s requests count.", subsys),
			},
			[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_code"},
		),
	}
}

func newSentBytes(subsys string, opts histogramOptions) observer {
	if opts.disable {
		return nil
	}
	typ := "responses"
	if subsys == "client" {
		typ = "requests"
	}
	if len(opts.buckets) > 0 {
		return &histogram{prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "sent_bytes",
				Help:      fmt.Sprintf("Bytes sent in gRPC %s %s.", subsys, typ),
				Buckets:   opts.buckets,
			},
			[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_frame"},
		)}
	}
	return &counters{
		sum: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "sent_bytes_sum",
				Help:      fmt.Sprintf("Bytes sent in gRPC %s %s sum.", subsys, typ),
			},
			[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_frame"},
		),
		num: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "sent_bytes_count",
				Help:      fmt.Sprintf("Bytes sent in gRPC %s %s count.", subsys, typ),
			},
			[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_frame"},
		),
	}
}

func newRecvBytes(subsys string, opts histogramOptions) observer {
	if opts.disable {
		return nil
	}
	typ := "requests"
	if subsys == "client" {
		typ = "responses"
	}
	if len(opts.buckets) > 0 {
		return &histogram{prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "recv_bytes",
				Help:      fmt.Sprintf("Bytes received in gRPC %s %s.", subsys, typ),
				Buckets:   opts.buckets,
			},
			[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_frame"},
		)}
	}
	return &counters{
		sum: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "recv_bytes_sum",
				Help:      fmt.Sprintf("Bytes received in gRPC %s %s sum.", subsys, typ),
			},
			[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_frame"},
		),
		num: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "grpc",
				Subsystem: subsys,
				Name:      "recv_bytes_count",
				Help:      fmt.Sprintf("Bytes received in gRPC %s %s count.", subsys, typ),
			},
			[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_frame"},
		),
	}
}

func (h *handler) init(server string, methods []grpc.MethodInfo, codes []codes.Code) {
	for _, meth := range methods {
		typ := grpcType(meth.IsClientStream, meth.IsServerStream)
		h.methods.Store("/"+server+"/"+meth.Name, methodInfo{
			typ:    typ,
			server: server,
			method: meth.Name,
		})
		if h.reqsPending != nil {
			h.reqsPending.GetMetricWithLabelValues(typ, server, meth.Name)
		}
		for _, c := range codes {
			code := c.String()
			if h.reqsTotal != nil {
				h.reqsTotal.GetMetricWithLabelValues(typ, server, meth.Name, code)
			}
			if h.latency != nil {
				h.latency.Init(typ, server, meth.Name, code)
			}
		}
		for _, f := range [...]string{header, payload, trailer} {
			if h.sentBytes != nil {
				h.sentBytes.Init(typ, server, meth.Name, f)
			}
			if h.recvBytes != nil {
				h.recvBytes.Init(typ, server, meth.Name, f)
			}
		}
	}

}

func (h *handler) collector() prometheus.Collector {
	// TODO: remove collector or describe/collect
	var c collectors
	if h.connsOpen != nil {
		c = append(c, h.connsOpen)
	}
	if h.connsTotal != nil {
		c = append(c, h.connsTotal)
	}
	if h.reqsPending != nil {
		c = append(c, h.reqsPending)
	}
	if h.reqsTotal != nil {
		c = append(c, h.reqsTotal)
	}
	if h.latency != nil {
		c = append(c, h.latency)
	}
	if h.sentBytes != nil {
		c = append(c, h.sentBytes)
	}
	if h.recvBytes != nil {
		c = append(c, h.recvBytes)
	}
	return c
}

func (h *handler) describe(ch chan<- *prometheus.Desc) {
	if h.connsOpen != nil {
		h.connsOpen.Describe(ch)
	}
	if h.connsTotal != nil {
		h.connsTotal.Describe(ch)
	}
	if h.reqsPending != nil {
		h.reqsPending.Describe(ch)
	}
	if h.reqsTotal != nil {
		h.reqsTotal.Describe(ch)
	}
	if h.latency != nil {
		h.latency.Describe(ch)
	}
	if h.sentBytes != nil {
		h.sentBytes.Describe(ch)
	}
	if h.recvBytes != nil {
		h.recvBytes.Describe(ch)
	}
}

func (h *handler) collect(ch chan<- prometheus.Metric) {
	if h.connsOpen != nil {
		h.connsOpen.Collect(ch)
	}
	if h.connsTotal != nil {
		h.connsTotal.Collect(ch)
	}
	if h.reqsPending != nil {
		h.reqsPending.Collect(ch)
	}
	if h.reqsTotal != nil {
		h.reqsTotal.Collect(ch)
	}
	if h.latency != nil {
		h.latency.Collect(ch)
	}
	if h.sentBytes != nil {
		h.sentBytes.Collect(ch)
	}
	if h.recvBytes != nil {
		h.recvBytes.Collect(ch)
	}
}

// TagConn implements the stats.Handler interface.
func (*handler) TagConn(ctx context.Context, v *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn implements the stats.Handler interface.
func (h *handler) HandleConn(ctx context.Context, stat stats.ConnStats) {
	switch stat.(type) {
	case *stats.ConnBegin:
		if h.connsOpen != nil {
			h.connsOpen.Inc()
		}
		if h.connsTotal != nil {
			h.connsTotal.Inc()
		}
	case *stats.ConnEnd:
		if h.connsOpen != nil {
			h.connsOpen.Dec()
		}
	}
}

var rpcInfoKey = "grpcprom"

type rpcInfo struct {
	methodInfo
	begin time.Time
}

type methodInfo struct {
	typ    string
	server string
	method string
}

func (h *handler) methodInfo(method, typ string) methodInfo {
	x, _ := h.methods.Load(method)
	if info, ok := x.(methodInfo); ok {
		return info
	}
	srv, meth := splitFullMethodName(method)
	info := methodInfo{
		typ:    typ,
		server: srv,
		method: meth,
	}
	if typ != unknown {
		h.methods.Store(method, info)
	}
	return info
}

// TagRPC implements the stats.Handler interface.
func (h *handler) TagRPC(ctx context.Context, v *stats.RPCTagInfo) context.Context {
	if _, ok := ctx.Value(&rpcInfoKey).(*rpcInfo); ok {
		return ctx
	}
	return context.WithValue(ctx, &rpcInfoKey, &rpcInfo{
		methodInfo: h.methodInfo(v.FullMethodName, unknown),
	})
}

func splitFullMethodName(s string) (server, method string) {
	s = strings.TrimPrefix(s, "/")
	i := strings.Index(s, "/")
	if i < 0 {
		return "Unknown", "Unknown"
	}
	return s[:i], s[i+1:]
}

// HandleRPC implements the stats.Handler interface.
func (h *handler) HandleRPC(ctx context.Context, stat stats.RPCStats) {
	v, ok := ctx.Value(&rpcInfoKey).(*rpcInfo)
	if !ok {
		return
	}
	switch s := stat.(type) {
	case *stats.Begin:
		v.begin = s.BeginTime
		if h.reqsPending != nil {
			h.reqsPending.WithLabelValues(v.typ, v.server, v.method).Inc()
		}
	case *stats.End:
		code := status.Code(s.Error).String()
		if h.latency != nil {
			h.latency.Observe(time.Since(v.begin).Seconds(), v.typ, v.server, v.method, code)
		}
		if h.reqsTotal != nil {
			h.reqsTotal.WithLabelValues(v.typ, v.server, v.method, code).Inc()
		}
		if h.reqsPending != nil {
			h.reqsPending.WithLabelValues(v.typ, v.server, v.method).Dec()
		}
	case *stats.InHeader:
		if h.recvBytes != nil {
			h.recvBytes.Observe(float64(s.WireLength), v.typ, v.server, v.method, header)
		}
	case *stats.InPayload:
		if h.recvBytes != nil {
			h.recvBytes.Observe(float64(s.WireLength), v.typ, v.server, v.method, payload)
		}
	case *stats.InTrailer:
		if h.recvBytes != nil {
			h.recvBytes.Observe(float64(s.WireLength), v.typ, v.server, v.method, trailer)
		}
	case *stats.OutHeader:
		if h.sentBytes != nil {
			// TODO: WireLength doesn't exist ???
			h.sentBytes.Observe(0, v.typ, v.server, v.method, header)
		}
	case *stats.OutPayload:
		if h.sentBytes != nil {
			h.sentBytes.Observe(float64(s.WireLength), v.typ, v.server, v.method, payload)
		}
	case *stats.OutTrailer:
		if h.sentBytes != nil {
			// TODO: WireLength is never set ???
			h.sentBytes.Observe(float64(s.WireLength), v.typ, v.server, v.method, trailer)
		}
	}
}

func (h *handler) unaryClientInterceptor(
	ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
) error {
	ctx = h.context(ctx, method, unary)
	return invoker(ctx, method, req, reply, cc, opts...)
}

func (h *handler) unaryServerInterceptor(
	ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	ctx = h.context(ctx, info.FullMethod, unary)
	return handler(ctx, req)
}

func (h *handler) streamClientInterceptor(
	ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	ctx = h.context(ctx, method, grpcType(desc.ClientStreams, desc.ServerStreams))
	return streamer(ctx, desc, cc, method, opts...)
}

func (h *handler) streamServerInterceptor(
	srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler,
) error {
	typ := grpcType(info.IsClientStream, info.IsServerStream)
	return handler(srv, &ctxServerStream{
		ServerStream: ss,
		ctx:          h.context(ss.Context(), info.FullMethod, typ),
	})
}

func (h *handler) context(ctx context.Context, method string, typ string) context.Context {
	info := h.methodInfo(method, typ)
	if v, ok := ctx.Value(&rpcInfoKey).(*rpcInfo); ok {
		v.methodInfo = info
		return ctx
	}
	return context.WithValue(ctx, &rpcInfoKey, &rpcInfo{methodInfo: info})
}

type ctxServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (ss *ctxServerStream) Context() context.Context {
	return ss.ctx
}

func grpcType(isClientStream, isServerStream bool) string {
	if isServerStream {
		if isClientStream {
			return bidiStream
		}
		return serverStream
	}
	if isClientStream {
		return clientStream
	}
	return unary
}
