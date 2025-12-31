package flags

import "fmt"

// ValidateConfidence validates that confidence is between 0.0 and 1.0.
func ValidateConfidence(v float64) error {
	if v < 0.0 || v > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0, got %f", v)
	}
	return nil
}

// ValidateLimit validates that limit is non-negative.
func ValidateLimit(v int) error {
	if v < 0 {
		return fmt.Errorf("limit must be non-negative, got %d", v)
	}
	return nil
}

// ValidateScope validates that scope is one of: file, room, palace.
func ValidateScope(v string) error {
	valid := map[string]bool{"file": true, "room": true, "palace": true}
	if !valid[v] {
		return fmt.Errorf("scope must be file, room, or palace, got %q", v)
	}
	return nil
}

// ValidatePort validates that port is between 1 and 65535.
func ValidatePort(v int) error {
	if v < 1 || v > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", v)
	}
	return nil
}
