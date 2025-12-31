package schemas

import (
	"bytes"
	"embed"
	"fmt"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed *.schema.json
var schemaFS embed.FS

var (
	compileOnce sync.Once
	compiler    *jsonschema.Compiler
	compileErr  error
)

func getCompiler() (*jsonschema.Compiler, error) {
	compileOnce.Do(func() {
		c := jsonschema.NewCompiler()
		for _, name := range []string{Palace, Room, Playbook, ContextPack, ChangeSignal, ProjectProfile, ScanSummary} {
			filePath := schemaPath(name)
			data, err := schemaFS.ReadFile(filePath)
			if err != nil {
				compileErr = fmt.Errorf("read schema %s: %w", name, err)
				return
			}
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
			if err != nil {
				compileErr = fmt.Errorf("decode schema %s: %w", name, err)
				return
			}
			if err := c.AddResource(schemaURL(name), doc); err != nil {
				compileErr = fmt.Errorf("register schema %s: %w", name, err)
				return
			}
		}
		compiler = c
	})
	return compiler, compileErr
}

const (
	Palace         = "palace"
	Room           = "room"
	Playbook       = "playbook"
	ContextPack    = "context-pack"
	ChangeSignal   = "change-signal"
	ProjectProfile = "project-profile"
	ScanSummary    = "scan"
)

func schemaPath(name string) string {
	return fmt.Sprintf("%s.schema.json", name)
}

func schemaURL(name string) string {
	return fmt.Sprintf("mem://schemas/%s.schema.json", name)
}

func Compile(name string) (*jsonschema.Schema, error) {
	c, err := getCompiler()
	if err != nil {
		return nil, err
	}
	s, err := c.Compile(schemaURL(name))
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", name, err)
	}
	return s, nil
}

func List() (map[string][]byte, error) {
	names := []string{Palace, Room, Playbook, ContextPack, ChangeSignal, ProjectProfile, ScanSummary}
	out := make(map[string][]byte, len(names))
	for _, n := range names {
		path := schemaPath(n)
		b, err := schemaFS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read schema %s: %w", n, err)
		}
		out[n] = b
	}
	return out, nil
}
