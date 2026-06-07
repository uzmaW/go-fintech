package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	customDBBuckets = []float64{
		0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
	}

	customRedisBuckets = []float64{
		0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1,
	}
)

type Metrics struct {
	TransactionsTotal    *prometheus.CounterVec
	TransactionsInFlight prometheus.Gauge
	TransactionDuration  *prometheus.HistogramVec

	KafkaMessagesProduced *prometheus.CounterVec
	KafkaMessagesConsumed *prometheus.CounterVec
	KafkaConsumerLag      *prometheus.GaugeVec

	DBQueryDuration *prometheus.HistogramVec

	RedisOperationDuration *prometheus.HistogramVec

	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestTotal    *prometheus.CounterVec

	ErrorsTotal *prometheus.CounterVec

	ActiveConnections prometheus.Gauge
}

func New(serviceName string) *Metrics {
	m := &Metrics{
		TransactionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "fintech",
				Subsystem:   "transactions",
				Name:        "total",
				Help:        "Total number of transactions processed.",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"status", "type"},
		),

		TransactionsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "fintech",
				Subsystem:   "transactions",
				Name:        "in_flight",
				Help:        "Number of transactions currently being processed.",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
		),

		TransactionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   "fintech",
				Subsystem:   "transactions",
				Name:        "duration_seconds",
				Help:        "Duration of transactions in seconds.",
				Buckets:     customDBBuckets,
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"operation"},
		),

		KafkaMessagesProduced: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "fintech",
				Subsystem: "kafka",
				Name:      "messages_produced_total",
				Help:      "Total number of Kafka messages produced.",
			},
			[]string{"topic", "service"},
		),

		KafkaMessagesConsumed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "fintech",
				Subsystem: "kafka",
				Name:      "messages_consumed_total",
				Help:      "Total number of Kafka messages consumed.",
			},
			[]string{"topic", "service"},
		),

		KafkaConsumerLag: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "fintech",
				Subsystem: "kafka",
				Name:      "consumer_lag",
				Help:      "Current consumer lag per topic and consumer group.",
			},
			[]string{"topic", "group"},
		),

		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "fintech",
				Subsystem: "db",
				Name:      "query_duration_seconds",
				Help:      "Duration of database queries in seconds.",
				Buckets:   customDBBuckets,
			},
			[]string{"operation"},
		),

		RedisOperationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "fintech",
				Subsystem: "redis",
				Name:      "operation_duration_seconds",
				Help:      "Duration of Redis operations in seconds.",
				Buckets:   customRedisBuckets,
			},
			[]string{"operation"},
		),

		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "fintech",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path", "status_code"},
		),

		HTTPRequestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "fintech",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests.",
			},
			[]string{"method", "path", "status_code"},
		),

		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "fintech",
				Name:        "errors_total",
				Help:        "Total number of errors by type.",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
			[]string{"type"},
		),

		ActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "fintech",
				Name:        "active_connections",
				Help:        "Number of active connections.",
				ConstLabels: prometheus.Labels{"service": serviceName},
			},
		),
	}

	reg := prometheus.DefaultRegisterer
	reg.MustRegister(
		m.TransactionsTotal,
		m.TransactionsInFlight,
		m.TransactionDuration,
		m.KafkaMessagesProduced,
		m.KafkaMessagesConsumed,
		m.KafkaConsumerLag,
		m.DBQueryDuration,
		m.RedisOperationDuration,
		m.HTTPRequestDuration,
		m.HTTPRequestTotal,
		m.ErrorsTotal,
		m.ActiveConnections,
	)

	return m
}

func Handler() http.Handler {
	return promhttp.Handler()
}

func (m *Metrics) IncTransactions(status, txType string) {
	m.TransactionsTotal.WithLabelValues(status, txType).Inc()
}

func (m *Metrics) ObserveDuration(operation string, d time.Duration) {
	m.TransactionDuration.WithLabelValues(operation).Observe(d.Seconds())
}

func (m *Metrics) IncKafkaProduced(topic string) {
	m.KafkaMessagesProduced.WithLabelValues(topic, "").Inc()
}

func (m *Metrics) IncKafkaConsumed(topic string) {
	m.KafkaMessagesConsumed.WithLabelValues(topic, "").Inc()
}

func (m *Metrics) SetConsumerLag(topic, group string, lag float64) {
	m.KafkaConsumerLag.WithLabelValues(topic, group).Set(lag)
}

func (m *Metrics) IncErrors(errType string) {
	m.ErrorsTotal.WithLabelValues(errType).Inc()
}

func (m *Metrics) IncInFlight() {
	m.TransactionsInFlight.Inc()
}

func (m *Metrics) DecInFlight() {
	m.TransactionsInFlight.Dec()
}

func (m *Metrics) ObserveHTTPRequest(method, path string, statusCode int, d time.Duration) {
	code := http.StatusText(statusCode)
	m.HTTPRequestDuration.WithLabelValues(method, path, code).Observe(d.Seconds())
	m.HTTPRequestTotal.WithLabelValues(method, path, code).Inc()
}

func (m *Metrics) ObserveDBQuery(operation string, d time.Duration) {
	m.DBQueryDuration.WithLabelValues(operation).Observe(d.Seconds())
}
