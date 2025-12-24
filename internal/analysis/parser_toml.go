package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/toml"
)

type TOMLParser struct {
	parser *sitter.Parser
}

func NewTOMLParser() *TOMLParser {
	p := sitter.NewParser()
	p.SetLanguage(toml.GetLanguage())
	return &TOMLParser{parser: p}
}

func (p *TOMLParser) Language() Language {
	return LangTOML
}

func (p *TOMLParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangTOML),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis, "")

	return analysis, nil
}

func (p *TOMLParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis, currentSection string) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "table":
			section := p.parseTable(child, content, analysis)
			currentSection = section

		case "table_array_element":
			section := p.parseTableArray(child, content, analysis)
			currentSection = section

		case "pair":
			p.parsePair(child, content, analysis, currentSection)
		}

		p.extractSymbols(child, content, analysis, currentSection)
	}
}

func (p *TOMLParser) parseTable(node *sitter.Node, content []byte, analysis *FileAnalysis) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "dotted_key" || child.Type() == "bare_key" || child.Type() == "quoted_key" {
			section := p.extractKeyName(child, content)
			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:      section,
				Kind:      KindClass,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Exported:  true,
			})
			return section
		}
	}

	return ""
}

func (p *TOMLParser) parseTableArray(node *sitter.Node, content []byte, analysis *FileAnalysis) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "dotted_key" || child.Type() == "bare_key" || child.Type() == "quoted_key" {
			section := p.extractKeyName(child, content)
			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:      section,
				Kind:      KindVariable,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Signature: "[[" + section + "]]",
				Exported:  true,
			})
			return section
		}
	}

	return ""
}

func (p *TOMLParser) parsePair(node *sitter.Node, content []byte, analysis *FileAnalysis, section string) {
	var key string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "dotted_key", "bare_key", "quoted_key":
			key = p.extractKeyName(child, content)
		}
	}

	if key == "" {
		return
	}

	fullKey := key
	if section != "" {
		fullKey = section + "." + key
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      fullKey,
		Kind:      KindProperty,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	})
}

func (p *TOMLParser) extractKeyName(node *sitter.Node, content []byte) string {
	switch node.Type() {
	case "bare_key":
		return node.Content(content)

	case "quoted_key":
		value := node.Content(content)
		return strings.Trim(value, "\"'")

	case "dotted_key":
		var parts []string
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && (child.Type() == "bare_key" || child.Type() == "quoted_key") {
				parts = append(parts, p.extractKeyName(child, content))
			}
		}
		return strings.Join(parts, ".")
	}

	return node.Content(content)
}
