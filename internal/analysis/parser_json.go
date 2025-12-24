package analysis

import (
	"encoding/json"
	"strings"
)

type JSONParser struct{}

func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

func (p *JSONParser) Language() Language {
	return LangJSON
}

func (p *JSONParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangJSON),
	}

	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return analysis, nil
	}

	p.extractSymbols(data, "", analysis, 1, len(strings.Split(string(content), "\n")))

	return analysis, nil
}

func (p *JSONParser) extractSymbols(data interface{}, prefix string, analysis *FileAnalysis, lineStart, lineEnd int) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			fullKey := key
			if prefix != "" {
				fullKey = prefix + "." + key
			}

			kind := KindProperty
			switch value.(type) {
			case map[string]interface{}:
				kind = KindClass
			case []interface{}:
				kind = KindVariable
			}

			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:      fullKey,
				Kind:      kind,
				LineStart: lineStart,
				LineEnd:   lineEnd,
				Exported:  true,
			})

			p.extractSymbols(value, fullKey, analysis, lineStart, lineEnd)
		}
	case []interface{}:
		for i, item := range v {
			if obj, ok := item.(map[string]interface{}); ok {
				itemPrefix := prefix
				if itemPrefix == "" {
					itemPrefix = "[" + string(rune('0'+i)) + "]"
				} else {
					itemPrefix = prefix + "[" + string(rune('0'+i)) + "]"
				}
				p.extractSymbols(obj, itemPrefix, analysis, lineStart, lineEnd)
			}
		}
	}
}
