package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/hcl"
)

type HCLParser struct {
	parser *sitter.Parser
}

func NewHCLParser() *HCLParser {
	p := sitter.NewParser()
	p.SetLanguage(hcl.GetLanguage())
	return &HCLParser{parser: p}
}

func (p *HCLParser) Language() Language {
	return LangHCL
}

func (p *HCLParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangHCL),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *HCLParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "block":
			sym := p.parseBlock(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "attribute":
			p.parseAttribute(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *HCLParser) parseBlock(node *sitter.Node, content []byte) *Symbol {
	var blockType string
	var labels []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "identifier":
			if blockType == "" {
				blockType = child.Content(content)
			}
		case "string_lit":
			label := strings.Trim(child.Content(content), "\"")
			labels = append(labels, label)
		}
	}

	if blockType == "" {
		return nil
	}

	name := blockType
	if len(labels) > 0 {
		name = strings.Join(labels, ".")
	}

	var kind SymbolKind
	switch blockType {
	case "resource":
		kind = KindClass
		if len(labels) >= 2 {
			name = labels[0] + "." + labels[1]
		}
	case "data":
		kind = KindType
		if len(labels) >= 2 {
			name = "data." + labels[0] + "." + labels[1]
		}
	case "module":
		kind = KindInterface
	case "variable":
		kind = KindVariable
	case "output":
		kind = KindProperty
	case "locals":
		kind = KindConstant
		name = "locals"
	case "provider":
		kind = KindType
	case "terraform":
		kind = KindType
		name = "terraform"
	default:
		kind = KindType
	}

	doc := p.extractPrecedingComment(node, content)
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "block_body" {
			children = p.extractBlockAttributes(child, content)
			break
		}
	}

	return &Symbol{
		Name:       name,
		Kind:       kind,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  blockType,
		DocComment: doc,
		Exported:   true,
		Children:   children,
	}
}

func (p *HCLParser) parseAttribute(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var name string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "identifier" {
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      name,
		Kind:      KindVariable,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	})
}

func (p *HCLParser) extractBlockAttributes(node *sitter.Node, content []byte) []Symbol {
	var attrs []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "attribute" {
			var name string
			for j := 0; j < int(child.ChildCount()); j++ {
				attr := child.Child(j)
				if attr != nil && attr.Type() == "identifier" {
					name = attr.Content(content)
					break
				}
			}

			if name != "" {
				attrs = append(attrs, Symbol{
					Name:      name,
					Kind:      KindProperty,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}
	return attrs
}

func (p *HCLParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "block":
			p.parseBlockReferences(child, content, analysis)

		case "expression":
			p.parseExpressionReferences(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *HCLParser) parseBlockReferences(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var blockType string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "identifier" {
			blockType = child.Content(content)
			break
		}
	}

	if blockType == "module" {
		p.parseModuleSource(node, content, analysis)
	}
}

func (p *HCLParser) parseModuleSource(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "block_body" {
			for j := 0; j < int(child.ChildCount()); j++ {
				attr := child.Child(j)
				if attr != nil && attr.Type() == "attribute" {
					var attrName string
					var attrValue string

					for k := 0; k < int(attr.ChildCount()); k++ {
						part := attr.Child(k)
						if part == nil {
							continue
						}

						if part.Type() == "identifier" {
							attrName = part.Content(content)
						}
						if part.Type() == "expression" {
							attrValue = strings.Trim(part.Content(content), "\"")
						}
					}

					if attrName == "source" && attrValue != "" {
						analysis.Relationships = append(analysis.Relationships, Relationship{
							TargetFile: attrValue,
							Kind:       RelImport,
							Line:       int(attr.StartPoint().Row) + 1,
						})
					}
				}
			}
		}
	}
}

func (p *HCLParser) parseExpressionReferences(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	text := node.Content(content)

	if strings.Contains(text, "var.") {
		parts := strings.Split(text, "var.")
		for i := 1; i < len(parts); i++ {
			varName := extractIdentifier(parts[i])
			if varName != "" {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetSymbol: "var." + varName,
					Kind:         RelReference,
					Line:         int(node.StartPoint().Row) + 1,
				})
			}
		}
	}

	if strings.Contains(text, "local.") {
		parts := strings.Split(text, "local.")
		for i := 1; i < len(parts); i++ {
			localName := extractIdentifier(parts[i])
			if localName != "" {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetSymbol: "local." + localName,
					Kind:         RelReference,
					Line:         int(node.StartPoint().Row) + 1,
				})
			}
		}
	}

	if strings.Contains(text, "module.") {
		parts := strings.Split(text, "module.")
		for i := 1; i < len(parts); i++ {
			moduleName := extractIdentifier(parts[i])
			if moduleName != "" {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetSymbol: "module." + moduleName,
					Kind:         RelReference,
					Line:         int(node.StartPoint().Row) + 1,
				})
			}
		}
	}
}

func extractIdentifier(s string) string {
	var id strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			id.WriteRune(c)
		} else {
			break
		}
	}
	return id.String()
}

func (p *HCLParser) extractPrecedingComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		comment := prev.Content(content)
		comment = strings.TrimPrefix(comment, "# ")
		comment = strings.TrimPrefix(comment, "// ")
		comment = strings.TrimPrefix(comment, "/* ")
		comment = strings.TrimSuffix(comment, " */")
		return comment
	}

	return ""
}
