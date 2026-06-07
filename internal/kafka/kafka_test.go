package kafka

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestNewProducer(t *testing.T) {
	cfg := ProducerConfig{
		Brokers:      []string{"localhost:9092"},
		Topic:        "test-topic",
		BatchSize:    100,
		BatchTimeout: time.Second,
		Compression:  kafka.Gzip,
	}

	producer := NewProducer(cfg)
	if producer == nil {
		t.Fatal("expected non-nil producer")
	}
	if producer.writer == nil {
		t.Fatal("expected non-nil writer")
	}
}

func TestProducerConfig(t *testing.T) {
	cfg := ProducerConfig{
		Brokers:      []string{"broker1:9092", "broker2:9092"},
		Topic:        "my-topic",
		Idempotent:   true,
		BatchSize:    500,
		BatchTimeout: 5 * time.Second,
		Compression:  kafka.Snappy,
		Async:        true,
		RequiredAcks: kafka.RequireOne,
	}

	if len(cfg.Brokers) != 2 {
		t.Errorf("Brokers len = %d, want 2", len(cfg.Brokers))
	}
	if cfg.Topic != "my-topic" {
		t.Errorf("Topic = %s, want my-topic", cfg.Topic)
	}
	if !cfg.Idempotent {
		t.Error("Idempotent = false, want true")
	}
	if cfg.BatchSize != 500 {
		t.Errorf("BatchSize = %d, want 500", cfg.BatchSize)
	}
	if cfg.BatchTimeout != 5*time.Second {
		t.Errorf("BatchTimeout = %v, want 5s", cfg.BatchTimeout)
	}
	if cfg.Compression != kafka.Snappy {
		t.Errorf("Compression = %v, want Snappy", cfg.Compression)
	}
	if !cfg.Async {
		t.Error("Async = false, want true")
	}
	if cfg.RequiredAcks != kafka.RequireOne {
		t.Errorf("RequiredAcks = %v, want RequireOne", cfg.RequiredAcks)
	}
}

func TestNewConsumer(t *testing.T) {
	cfg := ConsumerConfig{
		Brokers:        []string{"localhost:9092"},
		Topic:          "test-topic",
		GroupID:        "test-group",
		MinBytes:       1,
		MaxBytes:       1024,
		MaxWait:        time.Second,
		StartOffset:    kafka.FirstOffset,
		CommitInterval: time.Second,
	}

	consumer := NewConsumer(cfg)
	if consumer == nil {
		t.Fatal("expected non-nil consumer")
	}
	if consumer.reader == nil {
		t.Fatal("expected non-nil reader")
	}
}

func TestConsumerConfig(t *testing.T) {
	cfg := ConsumerConfig{
		Brokers:        []string{"broker1:9092", "broker2:9092", "broker3:9092"},
		Topic:          "events",
		GroupID:        "analytics-group",
		MinBytes:       10,
		MaxBytes:       4096,
		MaxWait:        3 * time.Second,
		StartOffset:    kafka.LastOffset,
		CommitInterval: 5 * time.Second,
	}

	if len(cfg.Brokers) != 3 {
		t.Errorf("Brokers len = %d, want 3", len(cfg.Brokers))
	}
	if cfg.Topic != "events" {
		t.Errorf("Topic = %s, want events", cfg.Topic)
	}
	if cfg.GroupID != "analytics-group" {
		t.Errorf("GroupID = %s, want analytics-group", cfg.GroupID)
	}
	if cfg.MinBytes != 10 {
		t.Errorf("MinBytes = %d, want 10", cfg.MinBytes)
	}
	if cfg.MaxBytes != 4096 {
		t.Errorf("MaxBytes = %d, want 4096", cfg.MaxBytes)
	}
	if cfg.MaxWait != 3*time.Second {
		t.Errorf("MaxWait = %v, want 3s", cfg.MaxWait)
	}
	if cfg.StartOffset != kafka.LastOffset {
		t.Errorf("StartOffset = %d, want LastOffset", cfg.StartOffset)
	}
	if cfg.CommitInterval != 5*time.Second {
		t.Errorf("CommitInterval = %v, want 5s", cfg.CommitInterval)
	}
}

func TestCompressionTypes(t *testing.T) {
	validTypes := map[kafka.Compression]bool{
		0:            true,
		kafka.Gzip:   true,
		kafka.Snappy: true,
		kafka.Lz4:    true,
		kafka.Zstd:   true,
	}

	for _, ct := range []kafka.Compression{0, kafka.Gzip, kafka.Snappy, kafka.Lz4, kafka.Zstd} {
		if !validTypes[ct] {
			t.Errorf("unexpected compression type: %v", ct)
		}
	}
}

func createTestSchemasDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	schema := `{
		"type": "record",
		"name": "TestRecord",
		"namespace": "com.test",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "amount", "type": "double"}
		]
	}`
	if err := os.WriteFile(filepath.Join(dir, "test_record.avsc"), []byte(schema), 0644); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}
	return dir
}

func TestNewAvroSerializer(t *testing.T) {
	dir := createTestSchemasDir(t)
	ser, err := NewAvroSerializer(dir)
	if err != nil {
		t.Fatalf("NewAvroSerializer error: %v", err)
	}
	if ser == nil {
		t.Fatal("expected non-nil serializer")
	}
}

func TestNewAvroSerializer_InvalidDir(t *testing.T) {
	_, err := NewAvroSerializer("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent dir")
	}
}

func TestAvroSerializer_Serialize(t *testing.T) {
	dir := createTestSchemasDir(t)
	ser, err := NewAvroSerializer(dir)
	if err != nil {
		t.Fatalf("NewAvroSerializer error: %v", err)
	}

	record := map[string]interface{}{
		"id":     "test-123",
		"amount": 99.95,
	}
	data, err := ser.Serialize("TestRecord", record)
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty serialized data")
	}
}

func TestAvroSerializer_RoundTrip(t *testing.T) {
	dir := createTestSchemasDir(t)
	ser, err := NewAvroSerializer(dir)
	if err != nil {
		t.Fatalf("NewAvroSerializer error: %v", err)
	}

	record := map[string]interface{}{
		"id":     "rt-456",
		"amount": 250.75,
	}
	data, err := ser.Serialize("TestRecord", record)
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}

	decoded, err := ser.Deserialize("TestRecord", data)
	if err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}

	native, ok := decoded.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", decoded)
	}

	if native["id"] != "rt-456" {
		t.Errorf("id mismatch: %v", native["id"])
	}
}

func TestAvroSerializer_Deserialize(t *testing.T) {
	dir := createTestSchemasDir(t)
	ser, err := NewAvroSerializer(dir)
	if err != nil {
		t.Fatalf("NewAvroSerializer error: %v", err)
	}

	record := map[string]interface{}{
		"id":     "deser-1",
		"amount": 42.0,
	}
	data, err := ser.Serialize("TestRecord", record)
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}

	decoded, err := ser.Deserialize("TestRecord", data)
	if err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}
	if decoded == nil {
		t.Fatal("expected non-nil decoded value")
	}
}

func TestAvroSerializer_SchemaNotFound(t *testing.T) {
	dir := createTestSchemasDir(t)
	ser, err := NewAvroSerializer(dir)
	if err != nil {
		t.Fatalf("NewAvroSerializer error: %v", err)
	}

	_, err = ser.Serialize("NonExistent", map[string]interface{}{"x": 1})
	if err == nil {
		t.Fatal("expected error for nonexistent schema")
	}
}

func TestAvroSerializer_DeserializeInvalidData(t *testing.T) {
	dir := createTestSchemasDir(t)
	ser, err := NewAvroSerializer(dir)
	if err != nil {
		t.Fatalf("NewAvroSerializer error: %v", err)
	}

	_, err = ser.Deserialize("TestRecord", []byte("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid data")
	}
}

func TestAvroSerializer_GetCodec(t *testing.T) {
	dir := createTestSchemasDir(t)
	ser, err := NewAvroSerializer(dir)
	if err != nil {
		t.Fatalf("NewAvroSerializer error: %v", err)
	}

	schema, err := ser.GetCodec("TestRecord")
	if err != nil {
		t.Fatalf("GetCodec error: %v", err)
	}
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}
	if schema.Name != "TestRecord" {
		t.Errorf("name mismatch: %s", schema.Name)
	}
	if schema.Codec == nil {
		t.Fatal("expected non-nil codec")
	}
}

func TestAvroSerializer_GetCodecNotFound(t *testing.T) {
	dir := createTestSchemasDir(t)
	ser, err := NewAvroSerializer(dir)
	if err != nil {
		t.Fatalf("NewAvroSerializer error: %v", err)
	}

	_, err = ser.GetCodec("NonExistent")
	if err == nil {
		t.Fatal("expected error for nonexistent schema")
	}
}

func TestAvroSerializer_MultipleRecords(t *testing.T) {
	dir := createTestSchemasDir(t)
	ser, err := NewAvroSerializer(dir)
	if err != nil {
		t.Fatalf("NewAvroSerializer error: %v", err)
	}

	records := []map[string]interface{}{
		{"id": "r1", "amount": 10.0},
		{"id": "r2", "amount": 20.5},
		{"id": "r3", "amount": 30.99},
	}

	for i, rec := range records {
		data, err := ser.Serialize("TestRecord", rec)
		if err != nil {
			t.Fatalf("Serialize %d error: %v", i, err)
		}
		decoded, err := ser.Deserialize("TestRecord", data)
		if err != nil {
			t.Fatalf("Deserialize %d error: %v", i, err)
		}
		native := decoded.(map[string]interface{})
		if native["id"] != rec["id"] {
			t.Errorf("record %d id mismatch: %v != %v", i, native["id"], rec["id"])
		}
	}
}
