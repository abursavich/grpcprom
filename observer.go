package grpcprom

import (
	"github.com/prometheus/client_golang/prometheus"
)

type observer interface {
	Init(lvs ...string)
	Observe(value float64, lvs ...string)
	prometheus.Collector
}

type histogram struct {
	m *prometheus.HistogramVec
}

func (h *histogram) Collect(ch chan<- prometheus.Metric) { h.m.Collect(ch) }
func (h *histogram) Describe(ch chan<- *prometheus.Desc) { h.m.Describe(ch) }
func (h *histogram) Observe(v float64, lvs ...string)    { h.m.WithLabelValues(lvs...).Observe(v) }
func (h *histogram) Init(lvs ...string)                  { h.m.GetMetricWithLabelValues(lvs...) }

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
