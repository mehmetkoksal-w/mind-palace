package jsonc

import (
	"encoding/json"
	"fmt"
	"os"

	jsonc "github.com/muhammadmuzzammil1998/jsonc"
)

// DecodeFile loads a JSONC file into the provided destination.
func DecodeFile(path string, dest any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	clean := jsonc.ToJSON(b)
	if err := json.Unmarshal(clean, dest); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// Clean strips comments and trailing commas from JSONC input.
func Clean(data []byte) []byte {
	return jsonc.ToJSON(data)
}
