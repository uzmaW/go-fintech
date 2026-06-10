package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry("/tmp/schemas")
	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if reg.schemas == nil {
		t.Fatal("schemas map is nil")
	}
	if reg.schemasDir != "/tmp/schemas" {
		t.Fatalf("expected schemasDir '/tmp/schemas', got %q", reg.schemasDir)
	}
}

func TestRegisterAndGetCodec(t *testing.T) {
	reg := NewRegistry("")
	schemaStr := `{"type":"record","name":"Test","namespace":"test","fields":[{"name":"id","type":"int"}]}`

	err := reg.RegisterSchema("test", "1", "avro", schemaStr)
	if err != nil {
		t.Fatalf("RegisterSchema failed: %v", err)
	}

	codec, err := reg.GetCodec("test")
	if err != nil {
		t.Fatalf("GetCodec failed: %v", err)
	}
	if codec == nil {
		t.Fatal("expected non-nil codec")
	}
}

func TestSerializeDeserializeAvro(t *testing.T) {
	reg := NewRegistry("")
	schemaStr := `{"type":"record","name":"Transaction","namespace":"fintech","fields":[{"name":"id","type":"int"},{"name":"amount","type":"long"},{"name":"currency","type":"string"}]}`

	if err := reg.RegisterSchema("transaction", "1", "avro", schemaStr); err != nil {
		t.Fatalf("RegisterSchema failed: %v", err)
	}

	record := map[string]interface{}{
		"id":       42,
		"amount":   int64(1500),
		"currency": "USD",
	}

	data, err := reg.SerializeAvro("transaction", record)
	if err != nil {
		t.Fatalf("SerializeAvro failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("serialized data is empty")
	}

	decoded, err := reg.DeserializeAvro("transaction", data)
	if err != nil {
		t.Fatalf("DeserializeAvro failed: %v", err)
	}

	native, ok := decoded.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", decoded)
	}

	if native["id"] != int32(42) {
		t.Errorf("expected id=42, got %v", native["id"])
	}
	if native["amount"] != int64(1500) {
		t.Errorf("expected amount=1500, got %v", native["amount"])
	}
	if native["currency"] != "USD" {
		t.Errorf("expected currency=USD, got %v", native["currency"])
	}
}

func TestLoadSchemasFromDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schema-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	schemaContent := `{"type":"record","name":"Payment","namespace":"fintech","fields":[{"name":"id","type":"int"}]}`
	if err := os.WriteFile(filepath.Join(tmpDir, "payment.avsc"), []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}

	reg := NewRegistry(tmpDir)
	if err := reg.LoadSchemasFromDir(tmpDir); err != nil {
		t.Fatalf("LoadSchemasFromDir failed: %v", err)
	}

	codec, err := reg.GetCodec("Payment")
	if err != nil {
		t.Fatalf("GetCodec failed after loading from dir: %v", err)
	}
	if codec == nil {
		t.Fatal("expected non-nil codec after loading from dir")
	}
}

func TestGetCodecNotFound(t *testing.T) {
	reg := NewRegistry("")
	_, err := reg.GetCodec("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent schema")
	}
}
