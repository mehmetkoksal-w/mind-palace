package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/rust"
)

type RustParser struct {
	parser *sitter.Parser
}

func NewRustParser() *RustParser {
	p := sitter.NewParser()
	p.SetLanguage(rust.GetLanguage())
	return &RustParser{parser: p}
}

func (p *RustParser) Language() Language {
	return LangRust
}

func (p *RustParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangRust),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *RustParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_item":
			sym := p.parseFunctionItem(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "struct_item":
			sym := p.parseStructItem(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "enum_item":
			sym := p.parseEnumItem(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "trait_item":
			sym := p.parseTraitItem(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "impl_item":
			p.parseImplItem(child, content, analysis)

		case "const_item", "static_item":
			sym := p.parseConstItem(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "type_item":
			sym := p.parseTypeItem(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *RustParser) parseFunctionItem(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	exported := p.hasVisibility(node, content)

	params := ""
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		params = paramsNode.Content(content)
	}

	returnType := ""
	returnNode := node.ChildByFieldName("return_type")
	if returnNode != nil {
		returnType = " -> " + returnNode.Content(content)
	}

	doc := p.extractDocComment(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindFunction,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  "fn " + name + params + returnType,
		DocComment: doc,
		Exported:   exported,
	}
}

func (p *RustParser) parseStructItem(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	exported := p.hasVisibility(node, content)
	doc := p.extractDocComment(node, content)

	sym := &Symbol{
		Name:       name,
		Kind:       KindClass,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   exported,
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		sym.Children = p.parseStructFields(bodyNode, content)
	}

	return sym
}

func (p *RustParser) parseStructFields(node *sitter.Node, content []byte) []Symbol {
	var fields []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil || child.Type() != "field_declaration" {
			continue
		}

		nameNode := child.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}

		fields = append(fields, Symbol{
			Name:      nameNode.Content(content),
			Kind:      KindProperty,
			LineStart: int(child.StartPoint().Row) + 1,
			LineEnd:   int(child.EndPoint().Row) + 1,
			Exported:  p.hasVisibility(child, content),
		})
	}

	return fields
}

func (p *RustParser) parseEnumItem(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	return &Symbol{
		Name:       nameNode.Content(content),
		Kind:       KindEnum,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: p.extractDocComment(node, content),
		Exported:   p.hasVisibility(node, content),
	}
}

func (p *RustParser) parseTraitItem(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	return &Symbol{
		Name:       nameNode.Content(content),
		Kind:       KindInterface,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: p.extractDocComment(node, content),
		Exported:   p.hasVisibility(node, content),
	}
}

func (p *RustParser) parseImplItem(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "declaration_list" {
			for j := 0; j < int(child.ChildCount()); j++ {
				item := child.Child(j)
				if item != nil && item.Type() == "function_item" {
					sym := p.parseFunctionItem(item, content)
					if sym != nil {
						sym.Kind = KindMethod
						analysis.Symbols = append(analysis.Symbols, *sym)
					}
				}
			}
		}
	}
}

func (p *RustParser) parseConstItem(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	return &Symbol{
		Name:      nameNode.Content(content),
		Kind:      KindConstant,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  p.hasVisibility(node, content),
	}
}

func (p *RustParser) parseTypeItem(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	return &Symbol{
		Name:      nameNode.Content(content),
		Kind:      KindType,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  p.hasVisibility(node, content),
	}
}

func (p *RustParser) hasVisibility(node *sitter.Node, _ []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "visibility_modifier" {
			return true
		}
	}
	return false
}

func (p *RustParser) extractDocComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "line_comment" {
		text := prev.Content(content)
		if strings.HasPrefix(text, "///") {
			return strings.TrimPrefix(text, "/// ")
		}
	}

	return ""
}

func (p *RustParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "use_declaration":
			p.parseUseDecl(child, content, analysis)

		case "call_expression":
			p.parseCallExpression(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *RustParser) parseCallExpression(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}

	var targetSymbol string

	switch funcNode.Type() {
	case "identifier":
		// Simple function call: foo()
		targetSymbol = funcNode.Content(content)

	case "scoped_identifier":
		// Qualified call: module::func() or Type::method()
		targetSymbol = funcNode.Content(content)

	case "field_expression":
		// Method call: obj.method()
		fieldNode := funcNode.ChildByFieldName("field")
		valueNode := funcNode.ChildByFieldName("value")
		if valueNode != nil && fieldNode != nil {
			targetSymbol = valueNode.Content(content) + "." + fieldNode.Content(content)
		} else if fieldNode != nil {
			targetSymbol = fieldNode.Content(content)
		}

	case "generic_function":
		// Generic function call: func::<T>()
		funcNameNode := funcNode.ChildByFieldName("function")
		if funcNameNode != nil {
			targetSymbol = funcNameNode.Content(content)
		}
	}

	if targetSymbol != "" {
		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetSymbol: targetSymbol,
			Kind:         RelCall,
			Line:         int(node.StartPoint().Row) + 1,
		})
	}
}

func (p *RustParser) parseUseDecl(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "scoped_identifier", "identifier", "use_wildcard", "use_list", "scoped_use_list":
			path := child.Content(content)
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: path,
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
		}
	}
}
