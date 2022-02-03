package grpcprom

// DefaultLatencyBuckets are the default latency histogram buckets.
var DefaultLatencyBuckets = []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// DefaultBytesBuckets are the default bytes histogram buckets.
var DefaultBytesBuckets = []float64{0, 32, 64, 128, 256, 512, 1024, 2048, 8192, 32768, 131072, 524288}

type metricOptions struct {
	disable bool
}

// A MetricOption applies an option to a metric.
type MetricOption interface {
	applyMetricOption(*metricOptions)
	HistogramOption
}

type metricOptionFunc func(*metricOptions)

func (fn metricOptionFunc) applyMetricOption(o *metricOptions)       { fn(o) }
func (fn metricOptionFunc) applyHistogramOption(o *histogramOptions) { fn(&o.metricOptions) }

// Disable returns a MetricOption that disables the metric.
func Disable() MetricOption {
	return metricOptionFunc(func(o *metricOptions) { o.disable = true })
}

type histogramOptions struct {
	metricOptions
	buckets []float64
}

// A HistogramOption applies an option to a histogram.
type HistogramOption interface{ applyHistogramOption(*histogramOptions) }

type histogramOptionFunc func(*histogramOptions)

func (fn histogramOptionFunc) applyHistogramOption(o *histogramOptions) { fn(o) }

// Buckets returns a HistogramOption that sets the histogram's buckets.
func Buckets(v []float64) HistogramOption {
	return histogramOptionFunc(func(o *histogramOptions) { o.buckets = v })
}

// NoBuckets returns a HistogramOption that disables the histogram's buckets.
// Instead, only sum and count will be collected.
func NoBuckets() HistogramOption {
	return histogramOptionFunc(func(o *histogramOptions) { o.buckets = nil })
}

type options struct {
	connsOpen   metricOptions
	connsTotal  metricOptions
	reqsPending metricOptions
	reqsTotal   metricOptions
	latency     histogramOptions
	recvBytes   histogramOptions
	sentBytes   histogramOptions
}

// An Option applies an option.
type Option interface {
	applyOption(*options)
}

type optionFunc func(*options)

func (fn optionFunc) applyOption(o *options) { fn(o) }

// ConnectionsOpen returns an Option that applies the given MetricOptions
// to the connections_open metric.
func ConnectionsOpen(opts ...MetricOption) Option {
	return optionFunc(func(o *options) {
		for _, opt := range opts {
			opt.applyMetricOption(&o.connsOpen)
		}
	})
}

// ConnectionsTotal returns an Option that applies the given MetricOptions
// to the connections_total metric.
func ConnectionsTotal(opts ...MetricOption) Option {
	return optionFunc(func(o *options) {
		for _, opt := range opts {
			opt.applyMetricOption(&o.connsTotal)
		}
	})
}

// RequestsPending returns an Option that applies the given MetricOptions
// to the requests_pending metric.
func RequestsPending(opts ...MetricOption) Option {
	return optionFunc(func(o *options) {
		for _, opt := range opts {
			opt.applyMetricOption(&o.reqsPending)
		}
	})
}

// RequestsTotal returns an Option that applies the given MetricOptions
// to the requests_total metric.
func RequestsTotal(opts ...MetricOption) Option {
	return optionFunc(func(o *options) {
		for _, opt := range opts {
			opt.applyMetricOption(&o.reqsTotal)
		}
	})
}

// LatencySeconds returns an Option that applies the given HistogramOption
// to the latency_seconds metric.
func LatencySeconds(opts ...HistogramOption) Option {
	return optionFunc(func(o *options) {
		for _, opt := range opts {
			opt.applyHistogramOption(&o.latency)
		}
	})
}

// RecvBytes returns an Option that applies the given HistogramOption
// to the recv_bytes metric.
func RecvBytes(opts ...HistogramOption) Option {
	return optionFunc(func(o *options) {
		for _, opt := range opts {
			opt.applyHistogramOption(&o.recvBytes)
		}
	})
}

// SentBytes returns an Option that applies the given HistogramOption
// to the sent_bytes metric.
func SentBytes(opts ...HistogramOption) Option {
	return optionFunc(func(o *options) {
		for _, opt := range opts {
			opt.applyHistogramOption(&o.sentBytes)
		}
	})
}
