package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/yaml"
)

type YAMLParser struct {
	parser *sitter.Parser
}

func NewYAMLParser() *YAMLParser {
	p := sitter.NewParser()
	p.SetLanguage(yaml.GetLanguage())
	return &YAMLParser{parser: p}
}

func (p *YAMLParser) Language() Language {
	return LangYAML
}

func (p *YAMLParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangYAML),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis, "")

	return analysis, nil
}

func (p *YAMLParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis, prefix string) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "block_mapping_pair":
			p.parseBlockMapping(child, content, analysis, prefix)

		case "flow_pair":
			p.parseFlowPair(child, content, analysis, prefix)

		case "block_mapping", "flow_mapping", "block_sequence", "flow_sequence":
			p.extractSymbols(child, content, analysis, prefix)
		}
	}
}

func (p *YAMLParser) parseBlockMapping(node *sitter.Node, content []byte, analysis *FileAnalysis, prefix string) {
	var key string
	var valueNode *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "flow_node", "block_node":
			if key == "" {
				key = p.extractScalarValue(child, content)
			} else {
				valueNode = child
			}
		}
	}

	if key == "" {
		return
	}

	fullKey := key
	if prefix != "" {
		fullKey = prefix + "." + key
	}

	kind := KindProperty
	if valueNode != nil {
		valueType := p.getValueType(valueNode)
		switch valueType {
		case "mapping":
			kind = KindClass
		case "sequence":
			kind = KindVariable
		}
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      fullKey,
		Kind:      kind,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	})

	if valueNode != nil {
		p.extractSymbols(valueNode, content, analysis, fullKey)
	}
}

func (p *YAMLParser) parseFlowPair(node *sitter.Node, content []byte, analysis *FileAnalysis, prefix string) {
	var key string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "flow_node" && key == "" {
			key = p.extractScalarValue(child, content)
			break
		}
	}

	if key == "" {
		return
	}

	fullKey := key
	if prefix != "" {
		fullKey = prefix + "." + key
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      fullKey,
		Kind:      KindProperty,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	})
}

func (p *YAMLParser) extractScalarValue(node *sitter.Node, content []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "plain_scalar", "single_quote_scalar", "double_quote_scalar":
			value := child.Content(content)
			value = strings.Trim(value, "\"'")
			return value
		case "flow_node", "block_node":
			return p.extractScalarValue(child, content)
		}
	}

	return ""
}

func (p *YAMLParser) getValueType(node *sitter.Node) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "block_mapping", "flow_mapping":
			return "mapping"
		case "block_sequence", "flow_sequence":
			return "sequence"
		case "block_node", "flow_node":
			return p.getValueType(child)
		}
	}

	return "scalar"
}
