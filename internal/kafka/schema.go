package kafka

import (
	"fmt"
	"log"

	"github.com/company/go-fintech/internal/schema"
)

// AvroSerializer provides Avro serialization for Kafka messages.
type AvroSerializer struct {
	registry *schema.SchemaRegistry
}

// NewAvroSerializer creates a serializer that loads schemas from the given directory.
func NewAvroSerializer(schemasDir string) (*AvroSerializer, error) {
	registry := schema.NewRegistry(schemasDir)
	if err := registry.LoadSchemasFromDir(schemasDir); err != nil {
		return nil, fmt.Errorf("failed to load Avro schemas from %s: %w", schemasDir, err)
	}
	log.Printf("Loaded Avro schemas from %s", schemasDir)
	return &AvroSerializer{registry: registry}, nil
}

// Serialize converts a Go value into Avro binary for the named schema.
func (s *AvroSerializer) Serialize(schemaName string, record interface{}) ([]byte, error) {
	return s.registry.SerializeAvro(schemaName, record)
}

// Deserialize decodes Avro binary into a native Go value.
func (s *AvroSerializer) Deserialize(schemaName string, data []byte) (interface{}, error) {
	return s.registry.DeserializeAvro(schemaName, data)
}

// GetCodec returns the goavro codec for manual use.
func (s *AvroSerializer) GetCodec(schemaName string) (*schema.Schema, error) {
	codec, err := s.registry.GetCodec(schemaName)
	if err != nil {
		return nil, err
	}
	return &schema.Schema{Name: schemaName, Codec: codec}, nil
}
