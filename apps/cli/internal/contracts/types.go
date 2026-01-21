// Package contracts provides FE-BE contract detection with type mismatch analysis.
package contracts

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// TypeSchema represents a type schema for comparing frontend and backend types.
// It uses a JSON Schema-like structure for type comparison.
type TypeSchema struct {
	Type       SchemaType             `json:"type"`                 // object, array, string, number, boolean, null, any
	Properties map[string]*TypeSchema `json:"properties,omitempty"` // For object types
	Items      *TypeSchema            `json:"items,omitempty"`      // For array types
	Required   []string               `json:"required,omitempty"`   // Required properties for objects
	Nullable   bool                   `json:"nullable,omitempty"`   // Can be null
	Enum       []string               `json:"enum,omitempty"`       // Allowed values
	Format     string                 `json:"format,omitempty"`     // Additional type info (date, email, uuid, etc.)
	Ref        string                 `json:"$ref,omitempty"`       // Reference to another type
}

// SchemaType represents the type of a schema field.
type SchemaType string

const (
	SchemaTypeObject  SchemaType = "object"
	SchemaTypeArray   SchemaType = "array"
	SchemaTypeString  SchemaType = "string"
	SchemaTypeNumber  SchemaType = "number"
	SchemaTypeInteger SchemaType = "integer"
	SchemaTypeBoolean SchemaType = "boolean"
	SchemaTypeNull    SchemaType = "null"
	SchemaTypeAny     SchemaType = "any"
	SchemaTypeUnknown SchemaType = "unknown"
)

// MismatchType represents the type of field mismatch between FE and BE.
type MismatchType string

const (
	MismatchMissingInFrontend   MismatchType = "missing_in_frontend"
	MismatchMissingInBackend    MismatchType = "missing_in_backend"
	MismatchTypeMismatch        MismatchType = "type_mismatch"
	MismatchOptionalityMismatch MismatchType = "optionality_mismatch"
	MismatchNullabilityMismatch MismatchType = "nullability_mismatch"
)

// MismatchSeverity indicates the severity of a mismatch.
type MismatchSeverity string

const (
	SeverityError   MismatchSeverity = "error"
	SeverityWarning MismatchSeverity = "warning"
	SeverityInfo    MismatchSeverity = "info"
)

// FieldMismatch represents a mismatch between frontend and backend for a specific field.
type FieldMismatch struct {
	ID           string           `json:"id"`
	FieldPath    string           `json:"field_path"` // e.g., "user.profile.email" or "data[].id"
	Type         MismatchType     `json:"type"`
	Severity     MismatchSeverity `json:"severity"`
	Description  string           `json:"description"`
	BackendType  string           `json:"backend_type,omitempty"`
	FrontendType string           `json:"frontend_type,omitempty"`
}

// NewObjectSchema creates a new object schema.
func NewObjectSchema() *TypeSchema {
	return &TypeSchema{
		Type:       SchemaTypeObject,
		Properties: make(map[string]*TypeSchema),
		Required:   []string{},
	}
}

// NewArraySchema creates a new array schema with item type.
func NewArraySchema(items *TypeSchema) *TypeSchema {
	return &TypeSchema{
		Type:  SchemaTypeArray,
		Items: items,
	}
}

// NewPrimitiveSchema creates a schema for a primitive type.
func NewPrimitiveSchema(t SchemaType) *TypeSchema {
	return &TypeSchema{Type: t}
}

// AddProperty adds a property to an object schema.
func (s *TypeSchema) AddProperty(name string, schema *TypeSchema, required bool) {
	if s.Properties == nil {
		s.Properties = make(map[string]*TypeSchema)
	}
	s.Properties[name] = schema
	if required {
		s.Required = append(s.Required, name)
	}
}

// IsRequired checks if a property is required.
func (s *TypeSchema) IsRequired(name string) bool {
	for _, r := range s.Required {
		if r == name {
			return true
		}
	}
	return false
}

// String returns a human-readable representation of the type.
func (s *TypeSchema) String() string {
	if s == nil {
		return "unknown"
	}

	switch s.Type {
	case SchemaTypeObject:
		if len(s.Properties) == 0 {
			return "object"
		}
		props := make([]string, 0, len(s.Properties))
		for name := range s.Properties {
			props = append(props, name)
		}
		sort.Strings(props)
		return fmt.Sprintf("object{%s}", strings.Join(props, ","))
	case SchemaTypeArray:
		if s.Items != nil {
			return fmt.Sprintf("array<%s>", s.Items.String())
		}
		return "array"
	default:
		str := string(s.Type)
		if s.Nullable {
			str += "?"
		}
		if s.Format != "" {
			str += fmt.Sprintf("(%s)", s.Format)
		}
		return str
	}
}

// Compare compares this schema with another and returns mismatches.
// backendSchema is "this", frontendSchema is "other".
func (s *TypeSchema) Compare(other *TypeSchema, path string) []FieldMismatch {
	if path == "" {
		path = "$"
	}

	var mismatches []FieldMismatch

	// Handle nil schemas
	if s == nil && other == nil {
		return mismatches
	}
	if s == nil {
		mismatches = append(mismatches, FieldMismatch{
			FieldPath:    path,
			Type:         MismatchMissingInBackend,
			Severity:     SeverityError,
			Description:  fmt.Sprintf("Field %q exists in frontend but not in backend", path),
			FrontendType: other.String(),
		})
		return mismatches
	}
	if other == nil {
		mismatches = append(mismatches, FieldMismatch{
			FieldPath:   path,
			Type:        MismatchMissingInFrontend,
			Severity:    SeverityWarning,
			Description: fmt.Sprintf("Field %q exists in backend but not used by frontend", path),
			BackendType: s.String(),
		})
		return mismatches
	}

	// Skip 'any' types - they match everything
	if s.Type == SchemaTypeAny || other.Type == SchemaTypeAny {
		return mismatches
	}

	// Check type mismatch (with compatible type handling)
	if !s.isTypeCompatible(other) {
		mismatches = append(mismatches, FieldMismatch{
			FieldPath:    path,
			Type:         MismatchTypeMismatch,
			Severity:     SeverityError,
			Description:  fmt.Sprintf("Type mismatch at %q: backend has %s, frontend expects %s", path, s.Type, other.Type),
			BackendType:  string(s.Type),
			FrontendType: string(other.Type),
		})
		return mismatches // Don't recurse on type mismatch
	}

	// Check nullability mismatch
	if s.Nullable && !other.Nullable {
		mismatches = append(mismatches, FieldMismatch{
			FieldPath:   path,
			Type:        MismatchNullabilityMismatch,
			Severity:    SeverityWarning,
			Description: fmt.Sprintf("Nullability mismatch at %q: backend allows null, frontend doesn't handle it", path),
			BackendType: s.String(),
		})
	}

	// Recursively compare based on type
	switch s.Type {
	case SchemaTypeObject:
		mismatches = append(mismatches, s.compareObjectProperties(other, path)...)
	case SchemaTypeArray:
		if s.Items != nil || other.Items != nil {
			itemPath := path + "[]"
			mismatches = append(mismatches, s.Items.Compare(other.Items, itemPath)...)
		}
	}

	return mismatches
}

// isTypeCompatible checks if two schema types are compatible.
func (s *TypeSchema) isTypeCompatible(other *TypeSchema) bool {
	if s.Type == other.Type {
		return true
	}

	// number and integer are compatible
	if (s.Type == SchemaTypeNumber && other.Type == SchemaTypeInteger) ||
		(s.Type == SchemaTypeInteger && other.Type == SchemaTypeNumber) {
		return true
	}

	return false
}

// compareObjectProperties compares object properties between backend and frontend schemas.
func (s *TypeSchema) compareObjectProperties(other *TypeSchema, path string) []FieldMismatch {
	var mismatches []FieldMismatch

	// Check backend properties exist in frontend
	for name, backendProp := range s.Properties {
		propPath := path + "." + name
		frontendProp := other.Properties[name]

		if frontendProp == nil {
			// Missing in frontend
			mismatches = append(mismatches, FieldMismatch{
				FieldPath:   propPath,
				Type:        MismatchMissingInFrontend,
				Severity:    SeverityWarning,
				Description: fmt.Sprintf("Backend property %q is not used by frontend", propPath),
				BackendType: backendProp.String(),
			})
			continue
		}

		// Recursively compare
		mismatches = append(mismatches, backendProp.Compare(frontendProp, propPath)...)

		// Check optionality
		backendRequired := s.IsRequired(name)
		frontendRequired := other.IsRequired(name)
		if !backendRequired && frontendRequired {
			mismatches = append(mismatches, FieldMismatch{
				FieldPath:   propPath,
				Type:        MismatchOptionalityMismatch,
				Severity:    SeverityWarning,
				Description: fmt.Sprintf("Optionality mismatch at %q: optional in backend, required in frontend", propPath),
			})
		}
	}

	// Check frontend properties that don't exist in backend
	for name, frontendProp := range other.Properties {
		propPath := path + "." + name
		if s.Properties[name] == nil {
			mismatches = append(mismatches, FieldMismatch{
				FieldPath:    propPath,
				Type:         MismatchMissingInBackend,
				Severity:     SeverityError,
				Description:  fmt.Sprintf("Frontend expects %q but backend doesn't provide it", propPath),
				FrontendType: frontendProp.String(),
			})
		}
	}

	return mismatches
}

// Clone creates a deep copy of the schema.
func (s *TypeSchema) Clone() *TypeSchema {
	if s == nil {
		return nil
	}

	clone := &TypeSchema{
		Type:     s.Type,
		Nullable: s.Nullable,
		Format:   s.Format,
		Ref:      s.Ref,
	}

	if len(s.Required) > 0 {
		clone.Required = make([]string, len(s.Required))
		copy(clone.Required, s.Required)
	}

	if len(s.Enum) > 0 {
		clone.Enum = make([]string, len(s.Enum))
		copy(clone.Enum, s.Enum)
	}

	if len(s.Properties) > 0 {
		clone.Properties = make(map[string]*TypeSchema, len(s.Properties))
		for k, v := range s.Properties {
			clone.Properties[k] = v.Clone()
		}
	}

	if s.Items != nil {
		clone.Items = s.Items.Clone()
	}

	return clone
}

// MarshalJSON implements json.Marshaler.
func (s *TypeSchema) MarshalJSON() ([]byte, error) {
	type Alias TypeSchema
	return json.Marshal((*Alias)(s))
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *TypeSchema) UnmarshalJSON(data []byte) error {
	type Alias TypeSchema
	return json.Unmarshal(data, (*Alias)(s))
}

// GoTypeToSchema maps a Go type name to a TypeSchema.
func GoTypeToSchema(goType string) *TypeSchema {
	// Handle pointers (nullable)
	nullable := false
	if strings.HasPrefix(goType, "*") {
		nullable = true
		goType = strings.TrimPrefix(goType, "*")
	}

	// Handle slices/arrays
	if strings.HasPrefix(goType, "[]") {
		itemType := strings.TrimPrefix(goType, "[]")
		return NewArraySchema(GoTypeToSchema(itemType))
	}

	// Handle maps
	if strings.HasPrefix(goType, "map[") {
		return &TypeSchema{Type: SchemaTypeObject, Nullable: nullable}
	}

	// Handle basic types
	var schema *TypeSchema
	switch goType {
	case "string":
		schema = NewPrimitiveSchema(SchemaTypeString)
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		schema = NewPrimitiveSchema(SchemaTypeInteger)
	case "float32", "float64":
		schema = NewPrimitiveSchema(SchemaTypeNumber)
	case "bool":
		schema = NewPrimitiveSchema(SchemaTypeBoolean)
	case "interface{}", "any":
		schema = NewPrimitiveSchema(SchemaTypeAny)
	case "time.Time":
		schema = &TypeSchema{Type: SchemaTypeString, Format: "date-time"}
	case "uuid.UUID":
		schema = &TypeSchema{Type: SchemaTypeString, Format: "uuid"}
	default:
		// Unknown type - treat as object
		schema = NewPrimitiveSchema(SchemaTypeUnknown)
	}

	schema.Nullable = nullable
	return schema
}

// TSTypeToSchema maps a TypeScript type to a TypeSchema.
func TSTypeToSchema(tsType string) *TypeSchema {
	// Handle arrays
	if strings.HasSuffix(tsType, "[]") {
		itemType := strings.TrimSuffix(tsType, "[]")
		return NewArraySchema(TSTypeToSchema(itemType))
	}
	if strings.HasPrefix(tsType, "Array<") && strings.HasSuffix(tsType, ">") {
		itemType := tsType[6 : len(tsType)-1]
		return NewArraySchema(TSTypeToSchema(itemType))
	}

	// Handle union with null/undefined (nullable)
	nullable := false
	if strings.Contains(tsType, "| null") || strings.Contains(tsType, "| undefined") {
		nullable = true
		tsType = strings.ReplaceAll(tsType, "| null", "")
		tsType = strings.ReplaceAll(tsType, "| undefined", "")
		tsType = strings.TrimSpace(tsType)
	}

	// Handle optional (question mark handled at property level)

	var schema *TypeSchema
	switch tsType {
	case "string":
		schema = NewPrimitiveSchema(SchemaTypeString)
	case "number":
		schema = NewPrimitiveSchema(SchemaTypeNumber)
	case "boolean":
		schema = NewPrimitiveSchema(SchemaTypeBoolean)
	case "null":
		schema = NewPrimitiveSchema(SchemaTypeNull)
	case "undefined":
		schema = NewPrimitiveSchema(SchemaTypeNull)
	case "any", "unknown":
		schema = NewPrimitiveSchema(SchemaTypeAny)
	case "object", "Object", "Record<string, any>":
		schema = NewObjectSchema()
	case "Date":
		schema = &TypeSchema{Type: SchemaTypeString, Format: "date-time"}
	default:
		// Unknown/custom type - could be interface
		schema = NewPrimitiveSchema(SchemaTypeUnknown)
	}

	schema.Nullable = nullable
	return schema
}
