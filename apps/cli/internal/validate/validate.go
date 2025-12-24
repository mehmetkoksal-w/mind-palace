package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/jsonc"
	"github.com/koksalmehmet/mind-palace/apps/cli/schemas"
)

// JSONC validates a JSONC file against an embedded schema.
func JSONC(path string, schemaName string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	clean := jsonc.Clean(data)
	schema, err := schemas.Compile(schemaName)
	if err != nil {
		return err
	}
	var instance any
	if err := json.Unmarshal(clean, &instance); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("%s invalid: %w", path, err)
	}
	return nil
}

// JSON validates a standard JSON file.
func JSON(path string, schemaName string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	schema, err := schemas.Compile(schemaName)
	if err != nil {
		return err
	}
	var instance any
	if err := json.Unmarshal(bytes.TrimSpace(data), &instance); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("%s invalid: %w", path, err)
	}
	return nil
}
