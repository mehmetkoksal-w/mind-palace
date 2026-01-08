package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/css"
)

type CSSParser struct {
	parser *sitter.Parser
}

func NewCSSParser() *CSSParser {
	p := sitter.NewParser()
	p.SetLanguage(css.GetLanguage())
	return &CSSParser{parser: p}
}

func (p *CSSParser) Language() Language {
	return LangCSS
}

func (p *CSSParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangCSS),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *CSSParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "rule_set":
			p.parseRuleSet(child, content, analysis)

		case "media_statement":
			sym := p.parseMediaQuery(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "keyframes_statement":
			sym := p.parseKeyframes(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "declaration":
			p.parseCustomProperty(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *CSSParser) parseRuleSet(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var selectors []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "selectors" {
			for j := 0; j < int(child.ChildCount()); j++ {
				sel := child.Child(j)
				if sel != nil {
					selectors = append(selectors, p.extractSelectorNames(sel, content)...)
				}
			}
		}
	}

	for _, selector := range selectors {
		kind := KindType
		switch {
		case strings.HasPrefix(selector, "#"):
			kind = KindVariable
		case strings.HasPrefix(selector, "."):
			kind = KindType
		case strings.HasPrefix(selector, "@"):
			kind = KindFunction
		}

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      selector,
			Kind:      kind,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
			Exported:  true,
		})
	}
}

func (p *CSSParser) extractSelectorNames(node *sitter.Node, content []byte) []string {
	var names []string

	switch node.Type() {
	case "class_selector":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "class_name" {
				names = append(names, "."+child.Content(content))
			}
		}

	case "id_selector":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "id_name" {
				names = append(names, "#"+child.Content(content))
			}
		}

	case "tag_name":
		names = append(names, node.Content(content))

	case "pseudo_class_selector", "pseudo_element_selector":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil {
				names = append(names, p.extractSelectorNames(child, content)...)
			}
		}

	default:
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil {
				names = append(names, p.extractSelectorNames(child, content)...)
			}
		}
	}

	return names
}

func (p *CSSParser) parseMediaQuery(node *sitter.Node, content []byte) *Symbol {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "media_query_list" {
			return &Symbol{
				Name:      "@media " + child.Content(content),
				Kind:      KindFunction,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Exported:  true,
			}
		}
	}

	return &Symbol{
		Name:      "@media",
		Kind:      KindFunction,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	}
}

func (p *CSSParser) parseKeyframes(node *sitter.Node, content []byte) *Symbol {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "keyframes_name" {
			return &Symbol{
				Name:      "@keyframes " + child.Content(content),
				Kind:      KindFunction,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Exported:  true,
			}
		}
	}

	return nil
}

func (p *CSSParser) parseCustomProperty(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "property_name" {
			propName := child.Content(content)
			if strings.HasPrefix(propName, "--") {
				analysis.Symbols = append(analysis.Symbols, Symbol{
					Name:      propName,
					Kind:      KindVariable,
					LineStart: int(node.StartPoint().Row) + 1,
					LineEnd:   int(node.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
			break
		}
	}
}

func (p *CSSParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_statement":
			p.parseImport(child, content, analysis)

		case "call_expression":
			p.parseUrlCall(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *CSSParser) parseImport(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "string_value" || child.Type() == "call_expression" {
			value := child.Content(content)
			value = strings.Trim(value, "\"'")
			if strings.HasPrefix(value, "url(") {
				value = strings.TrimPrefix(value, "url(")
				value = strings.TrimSuffix(value, ")")
				value = strings.Trim(value, "\"'")
			}

			if value != "" {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: value,
					Kind:       RelImport,
					Line:       int(node.StartPoint().Row) + 1,
				})
			}
			break
		}
	}
}

func (p *CSSParser) parseUrlCall(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	nodeContent := node.Content(content)
	if !strings.HasPrefix(nodeContent, "url(") {
		return
	}

	value := strings.TrimPrefix(nodeContent, "url(")
	value = strings.TrimSuffix(value, ")")
	value = strings.Trim(value, "\"'")

	if value != "" && !strings.HasPrefix(value, "data:") {
		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetFile: value,
			Kind:       RelReference,
			Line:       int(node.StartPoint().Row) + 1,
		})
	}
}
