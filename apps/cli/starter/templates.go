// Package starter provides embedded templates for palace init.
package starter

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed palace.jsonc project-profile.json rooms/*.jsonc playbooks/*.jsonc outputs/*.json
var templateFS embed.FS

// Get returns the template content for a relative path within starter/.
func Get(name string) (string, error) {
	path := strings.TrimPrefix(name, "/")
	data, err := templateFS.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Apply replaces placeholder keys with provided values in the template content.
func Apply(template string, replacements map[string]string) string {
	out := template
	for k, v := range replacements {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{%s}}", k), v)
	}
	return out
}
