package metrics

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

var sharedMetrics *Metrics

func TestMain(testingM *testing.M) {
	sharedMetrics = New("test-service")
	os.Exit(testingM.Run())
}

func TestNewMetrics(t *testing.T) {
	if sharedMetrics.TransactionsTotal == nil {
		t.Fatal("TransactionsTotal is nil")
	}
	if sharedMetrics.TransactionsInFlight == nil {
		t.Fatal("TransactionsInFlight is nil")
	}
	if sharedMetrics.TransactionDuration == nil {
		t.Fatal("TransactionDuration is nil")
	}
	if sharedMetrics.KafkaMessagesProduced == nil {
		t.Fatal("KafkaMessagesProduced is nil")
	}
	if sharedMetrics.KafkaMessagesConsumed == nil {
		t.Fatal("KafkaMessagesConsumed is nil")
	}
	if sharedMetrics.KafkaConsumerLag == nil {
		t.Fatal("KafkaConsumerLag is nil")
	}
	if sharedMetrics.DBQueryDuration == nil {
		t.Fatal("DBQueryDuration is nil")
	}
	if sharedMetrics.RedisOperationDuration == nil {
		t.Fatal("RedisOperationDuration is nil")
	}
	if sharedMetrics.HTTPRequestDuration == nil {
		t.Fatal("HTTPRequestDuration is nil")
	}
	if sharedMetrics.HTTPRequestTotal == nil {
		t.Fatal("HTTPRequestTotal is nil")
	}
	if sharedMetrics.ErrorsTotal == nil {
		t.Fatal("ErrorsTotal is nil")
	}
	if sharedMetrics.ActiveConnections == nil {
		t.Fatal("ActiveConnections is nil")
	}

	type collectorInfo struct {
		collector prometheus.Collector
		name      string
	}

	collectors := []collectorInfo{
		{sharedMetrics.TransactionsTotal, "TransactionsTotal"},
		{sharedMetrics.TransactionsInFlight, "TransactionsInFlight"},
		{sharedMetrics.TransactionDuration, "TransactionDuration"},
		{sharedMetrics.KafkaMessagesProduced, "KafkaMessagesProduced"},
		{sharedMetrics.KafkaMessagesConsumed, "KafkaMessagesConsumed"},
		{sharedMetrics.KafkaConsumerLag, "KafkaConsumerLag"},
		{sharedMetrics.DBQueryDuration, "DBQueryDuration"},
		{sharedMetrics.RedisOperationDuration, "RedisOperationDuration"},
		{sharedMetrics.HTTPRequestDuration, "HTTPRequestDuration"},
		{sharedMetrics.HTTPRequestTotal, "HTTPRequestTotal"},
		{sharedMetrics.ErrorsTotal, "ErrorsTotal"},
		{sharedMetrics.ActiveConnections, "ActiveConnections"},
	}

	for _, c := range collectors {
		count := testutil.CollectAndCount(c.collector)
		if count < 0 {
			t.Errorf("collector %s returned negative count: %d", c.name, count)
		}
	}
}

func TestIncTransactions(t *testing.T) {
	sharedMetrics.IncTransactions("success", "payment")
	sharedMetrics.IncTransactions("success", "payment")
	sharedMetrics.IncTransactions("failed", "transfer")

	expected := `
# HELP fintech_transactions_total Total number of transactions processed.
# TYPE fintech_transactions_total counter
fintech_transactions_total{service="test-service",status="failed",type="transfer"} 1
fintech_transactions_total{service="test-service",status="success",type="payment"} 2
`
	if err := testutil.CollectAndCompare(sharedMetrics.TransactionsTotal, strings.NewReader(expected)); err != nil {
		t.Errorf("unexpected metrics:\n%s", err)
	}
}

func TestObserveDuration(t *testing.T) {
	sharedMetrics.ObserveDuration("duration_test_op", 100*time.Millisecond)
	sharedMetrics.ObserveDuration("duration_test_op", 200*time.Millisecond)

	expected := `
# HELP fintech_transactions_duration_seconds Duration of transactions in seconds.
# TYPE fintech_transactions_duration_seconds histogram
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="0.005"} 0
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="0.01"} 0
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="0.025"} 0
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="0.05"} 0
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="0.1"} 1
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="0.25"} 2
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="0.5"} 2
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="1"} 2
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="2.5"} 2
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="5"} 2
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="10"} 2
fintech_transactions_duration_seconds_bucket{operation="duration_test_op",service="test-service",le="+Inf"} 2
fintech_transactions_duration_seconds_sum{operation="duration_test_op",service="test-service"} 0.30000000000000004
fintech_transactions_duration_seconds_count{operation="duration_test_op",service="test-service"} 2
`
	if err := testutil.CollectAndCompare(sharedMetrics.TransactionDuration, strings.NewReader(expected)); err != nil {
		t.Errorf("unexpected metrics:\n%s", err)
	}
}

func TestIncKafkaProduced(t *testing.T) {
	sharedMetrics.IncKafkaProduced("orders")
	sharedMetrics.IncKafkaProduced("orders")
	sharedMetrics.IncKafkaProduced("orders")

	expected := `
# HELP fintech_kafka_messages_produced_total Total number of Kafka messages produced.
# TYPE fintech_kafka_messages_produced_total counter
fintech_kafka_messages_produced_total{service="",topic="orders"} 3
`
	if err := testutil.CollectAndCompare(sharedMetrics.KafkaMessagesProduced, strings.NewReader(expected)); err != nil {
		t.Errorf("unexpected metrics:\n%s", err)
	}
}

func TestIncKafkaConsumed(t *testing.T) {
	sharedMetrics.IncKafkaConsumed("events")
	sharedMetrics.IncKafkaConsumed("events")

	expected := `
# HELP fintech_kafka_messages_consumed_total Total number of Kafka messages consumed.
# TYPE fintech_kafka_messages_consumed_total counter
fintech_kafka_messages_consumed_total{service="",topic="events"} 2
`
	if err := testutil.CollectAndCompare(sharedMetrics.KafkaMessagesConsumed, strings.NewReader(expected)); err != nil {
		t.Errorf("unexpected metrics:\n%s", err)
	}
}

func TestSetConsumerLag(t *testing.T) {
	sharedMetrics.SetConsumerLag("orders", "group-a", 42)
	sharedMetrics.SetConsumerLag("orders", "group-a", 10)

	expected := `
# HELP fintech_kafka_consumer_lag Current consumer lag per topic and consumer group.
# TYPE fintech_kafka_consumer_lag gauge
fintech_kafka_consumer_lag{group="group-a",topic="orders"} 10
`
	if err := testutil.CollectAndCompare(sharedMetrics.KafkaConsumerLag, strings.NewReader(expected)); err != nil {
		t.Errorf("unexpected metrics:\n%s", err)
	}
}

func TestIncErrors(t *testing.T) {
	sharedMetrics.IncErrors("timeout")
	sharedMetrics.IncErrors("timeout")
	sharedMetrics.IncErrors("connection")

	expected := `
# HELP fintech_errors_total Total number of errors by type.
# TYPE fintech_errors_total counter
fintech_errors_total{service="test-service",type="connection"} 1
fintech_errors_total{service="test-service",type="timeout"} 2
`
	if err := testutil.CollectAndCompare(sharedMetrics.ErrorsTotal, strings.NewReader(expected)); err != nil {
		t.Errorf("unexpected metrics:\n%s", err)
	}
}

func TestIncDecInFlight(t *testing.T) {
	sharedMetrics.IncInFlight()
	sharedMetrics.IncInFlight()
	sharedMetrics.IncInFlight()

	expected := `
# HELP fintech_transactions_in_flight Number of transactions currently being processed.
# TYPE fintech_transactions_in_flight gauge
fintech_transactions_in_flight{service="test-service"} 3
`
	if err := testutil.CollectAndCompare(sharedMetrics.TransactionsInFlight, strings.NewReader(expected)); err != nil {
		t.Errorf("after inc:\n%s", err)
	}

	sharedMetrics.DecInFlight()
	sharedMetrics.DecInFlight()

	expected = `
# HELP fintech_transactions_in_flight Number of transactions currently being processed.
# TYPE fintech_transactions_in_flight gauge
fintech_transactions_in_flight{service="test-service"} 1
`
	if err := testutil.CollectAndCompare(sharedMetrics.TransactionsInFlight, strings.NewReader(expected)); err != nil {
		t.Errorf("after dec:\n%s", err)
	}
}

func TestHandler(t *testing.T) {
	handler := Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "fintech_") {
		t.Error("response body does not contain fintech_ metrics")
	}
}

func TestPrometheusMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(PrometheusMiddleware(sharedMetrics))

	router.GET("/middleware_test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	startCount := testutil.ToFloat64(sharedMetrics.HTTPRequestTotal.WithLabelValues("GET", "/middleware_test", "OK"))

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/middleware_test", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, w.Code)
		}
	}

	endCount := testutil.ToFloat64(sharedMetrics.HTTPRequestTotal.WithLabelValues("GET", "/middleware_test", "OK"))
	if endCount-startCount != 5 {
		t.Errorf("expected 5 new requests recorded, got %f (start=%f, end=%f)", endCount-startCount, startCount, endCount)
	}

	if testutil.CollectAndCount(sharedMetrics.HTTPRequestDuration, "fintech_http_request_duration_seconds") == 0 {
		t.Error("expected HTTP request duration histogram to have metric families")
	}
}

func TestHTTPMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(PrometheusMiddleware(sharedMetrics))

	router.GET("/http_metrics_users", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	router.POST("/http_metrics_users", func(c *gin.Context) {
		c.String(http.StatusCreated, "created")
	})
	router.GET("/http_metrics_not_found", func(c *gin.Context) {
		c.String(http.StatusNotFound, "not found")
	})

	tests := []struct {
		method string
		path   string
		labels prometheus.Labels
	}{
		{http.MethodGet, "/http_metrics_users", prometheus.Labels{"method": "GET", "path": "/http_metrics_users", "status_code": "OK"}},
		{http.MethodPost, "/http_metrics_users", prometheus.Labels{"method": "POST", "path": "/http_metrics_users", "status_code": "Created"}},
		{http.MethodGet, "/http_metrics_not_found", prometheus.Labels{"method": "GET", "path": "/http_metrics_not_found", "status_code": "Not Found"}},
	}

	for _, tt := range tests {
		startCount := testutil.ToFloat64(sharedMetrics.HTTPRequestTotal.With(tt.labels))
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tt.method, tt.path, nil)
		router.ServeHTTP(w, req)
		endCount := testutil.ToFloat64(sharedMetrics.HTTPRequestTotal.With(tt.labels))
		if endCount-startCount != 1 {
			t.Errorf("expected %s %s count delta 1, got %f", tt.method, tt.path, endCount-startCount)
		}
	}
}

func TestObserveHTTPRequest(t *testing.T) {
	startCount := testutil.ToFloat64(sharedMetrics.HTTPRequestTotal.WithLabelValues("GET", "/api/v1/users", "OK"))
	sharedMetrics.ObserveHTTPRequest("GET", "/api/v1/users", http.StatusOK, 50*time.Millisecond)
	sharedMetrics.ObserveHTTPRequest("GET", "/api/v1/users", http.StatusOK, 150*time.Millisecond)
	endCount := testutil.ToFloat64(sharedMetrics.HTTPRequestTotal.WithLabelValues("GET", "/api/v1/users", "OK"))
	if endCount-startCount != 2 {
		t.Errorf("expected GET /api/v1/users count delta 2, got %f", endCount-startCount)
	}

	startCount = testutil.ToFloat64(sharedMetrics.HTTPRequestTotal.WithLabelValues("POST", "/api/v1/users", "Created"))
	sharedMetrics.ObserveHTTPRequest("POST", "/api/v1/users", http.StatusCreated, 250*time.Millisecond)
	endCount = testutil.ToFloat64(sharedMetrics.HTTPRequestTotal.WithLabelValues("POST", "/api/v1/users", "Created"))
	if endCount-startCount != 1 {
		t.Errorf("expected POST /api/v1/users count delta 1, got %f", endCount-startCount)
	}
}

func TestObserveDBQuery(t *testing.T) {
	startCount := testutil.CollectAndCount(sharedMetrics.DBQueryDuration, "fintech_db_query_duration_seconds")
	sharedMetrics.ObserveDBQuery("select_test", 10*time.Millisecond)
	sharedMetrics.ObserveDBQuery("select_test", 20*time.Millisecond)
	endCount := testutil.CollectAndCount(sharedMetrics.DBQueryDuration, "fintech_db_query_duration_seconds")

	if endCount <= startCount {
		t.Errorf("expected metric count to increase after observations, start=%d, end=%d", startCount, endCount)
	}
}

func TestDBQueryTracker(t *testing.T) {
	called := false
	tracker := DBQueryTracker(sharedMetrics)
	err := tracker("insert_test", func() error {
		called = true
		return nil
	})

	if !called {
		t.Error("tracked function was not called")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errTracker := DBQueryTracker(sharedMetrics)
	dbErr := errTracker("update_test", func() error {
		return http.ErrAbortHandler
	})
	if dbErr == nil {
		t.Fatal("expected error from tracked function")
	}

	errorCount := testutil.ToFloat64(sharedMetrics.ErrorsTotal.WithLabelValues("db_update_test"))
	if errorCount != 1 {
		t.Errorf("expected 1 error count, got %f", errorCount)
	}
}

func TestStatusFromInt(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "200"},
		{201, "201"},
		{404, "404"},
		{500, "500"},
	}

	for _, tt := range tests {
		result := StatusFromInt(tt.code)
		if result != tt.expected {
			t.Errorf("StatusFromInt(%d) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}
