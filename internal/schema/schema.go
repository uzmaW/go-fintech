package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/linkedin/goavro/v2"
)

type Schema struct {
	Name       string
	Version    int
	SchemaType string
	SchemaStr  string
	Codec      *goavro.Codec
}

type SchemaRegistry struct {
	schemas    map[string]*Schema
	mu         sync.RWMutex
	schemasDir string
}

func NewRegistry(schemasDir string) *SchemaRegistry {
	return &SchemaRegistry{
		schemas:    make(map[string]*Schema),
		schemasDir: schemasDir,
	}
}

func (r *SchemaRegistry) RegisterSchema(name, version, schemaType, schemaStr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	schema := &Schema{
		Name:       name,
		Version:    0,
		SchemaType: schemaType,
		SchemaStr:  schemaStr,
	}

	if schemaType == "avro" {
		codec, err := goavro.NewCodec(schemaStr)
		if err != nil {
			return fmt.Errorf("failed to parse avro schema: %w", err)
		}
		schema.Codec = codec
	}

	r.schemas[name] = schema
	return nil
}

func (r *SchemaRegistry) GetCodec(name string) (*goavro.Codec, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, ok := r.schemas[name]
	if !ok {
		return nil, fmt.Errorf("schema not found: %s", name)
	}

	if schema.Codec == nil {
		return nil, fmt.Errorf("schema %s does not have an avro codec", name)
	}

	return schema.Codec, nil
}

func (r *SchemaRegistry) SerializeAvro(name string, record interface{}) ([]byte, error) {
	codec, err := r.GetCodec(name)
	if err != nil {
		return nil, err
	}

	data, err := codec.BinaryFromNative(nil, record)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize avro record: %w", err)
	}

	return data, nil
}

func (r *SchemaRegistry) DeserializeAvro(name string, data []byte) (interface{}, error) {
	codec, err := r.GetCodec(name)
	if err != nil {
		return nil, err
	}

	native, _, err := codec.NativeFromBinary(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize avro record: %w", err)
	}

	return native, nil
}

func (r *SchemaRegistry) LoadSchemasFromDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(path) != ".avsc" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", path, err)
		}

		var schemaDef struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &schemaDef); err != nil {
			return fmt.Errorf("failed to parse schema file %s: %w", path, err)
		}

		if err := r.RegisterSchema(schemaDef.Name, "1", "avro", string(data)); err != nil {
			return fmt.Errorf("failed to register schema %s: %w", schemaDef.Name, err)
		}

		return nil
	})
}
