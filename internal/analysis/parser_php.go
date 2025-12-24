package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/php"
)

type PHPParser struct {
	parser *sitter.Parser
}

func NewPHPParser() *PHPParser {
	p := sitter.NewParser()
	p.SetLanguage(php.GetLanguage())
	return &PHPParser{parser: p}
}

func (p *PHPParser) Language() Language {
	return LangPHP
}

func (p *PHPParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangPHP),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *PHPParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "class_declaration":
			sym := p.parseClass(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "interface_declaration":
			sym := p.parseInterface(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "trait_declaration":
			sym := p.parseTrait(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "function_definition":
			sym := p.parseFunction(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "const_declaration":
			p.parseConstDecl(child, content, analysis)

		case "namespace_definition":
			p.parseNamespace(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *PHPParser) parseClass(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocComment(node, content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body != nil {
		children = p.extractClassMembers(body, content)
	}

	return &Symbol{
		Name:       name,
		Kind:       KindClass,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
		Children:   children,
	}
}

func (p *PHPParser) parseInterface(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocComment(node, content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body != nil {
		children = p.extractInterfaceMembers(body, content)
	}

	return &Symbol{
		Name:       name,
		Kind:       KindInterface,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
		Children:   children,
	}
}

func (p *PHPParser) parseTrait(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocComment(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindInterface,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
	}
}

func (p *PHPParser) parseFunction(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocComment(node, content)
	sig := p.extractFunctionSignature(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindFunction,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  sig,
		DocComment: doc,
		Exported:   true,
	}
}

func (p *PHPParser) parseConstDecl(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "const_element" {
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				analysis.Symbols = append(analysis.Symbols, Symbol{
					Name:      nameNode.Content(content),
					Kind:      KindConstant,
					LineStart: int(node.StartPoint().Row) + 1,
					LineEnd:   int(node.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}
}

func (p *PHPParser) parseNamespace(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      nameNode.Content(content),
		Kind:      KindType,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	})
}

func (p *PHPParser) extractClassMembers(node *sitter.Node, content []byte) []Symbol {
	var members []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "method_declaration":
			sym := p.parseMethod(child, content)
			if sym != nil {
				members = append(members, *sym)
			}

		case "property_declaration":
			p.parseProperties(child, content, &members)

		case "const_declaration":
			for j := 0; j < int(child.ChildCount()); j++ {
				elem := child.Child(j)
				if elem != nil && elem.Type() == "const_element" {
					nameNode := elem.ChildByFieldName("name")
					if nameNode != nil {
						members = append(members, Symbol{
							Name:      nameNode.Content(content),
							Kind:      KindConstant,
							LineStart: int(child.StartPoint().Row) + 1,
							LineEnd:   int(child.EndPoint().Row) + 1,
							Exported:  true,
						})
					}
				}
			}
		}
	}
	return members
}

func (p *PHPParser) parseMethod(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocComment(node, content)
	sig := p.extractMethodSignature(node, content)
	exported := p.isPublic(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindMethod,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  sig,
		DocComment: doc,
		Exported:   exported,
	}
}

func (p *PHPParser) parseProperties(node *sitter.Node, content []byte, members *[]Symbol) {
	exported := p.isPublic(node, content)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "property_element" {
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(content)
				name = strings.TrimPrefix(name, "$")
				*members = append(*members, Symbol{
					Name:      name,
					Kind:      KindProperty,
					LineStart: int(node.StartPoint().Row) + 1,
					LineEnd:   int(node.EndPoint().Row) + 1,
					Exported:  exported,
				})
			}
		}
	}
}

func (p *PHPParser) extractInterfaceMembers(node *sitter.Node, content []byte) []Symbol {
	var members []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "method_declaration" {
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				members = append(members, Symbol{
					Name:      nameNode.Content(content),
					Kind:      KindMethod,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}
	return members
}

func (p *PHPParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "namespace_use_declaration":
			p.parseUse(child, content, analysis)

		case "class_declaration":
			p.parseClassRelationships(child, content, analysis)

		case "interface_declaration":
			p.parseInterfaceExtends(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *PHPParser) parseUse(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "namespace_use_clause" || child.Type() == "qualified_name") {
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: child.Content(content),
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
		}
	}
}

func (p *PHPParser) parseClassRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	baseClause := node.ChildByFieldName("base_clause")
	if baseClause != nil {
		for i := 0; i < int(baseClause.ChildCount()); i++ {
			child := baseClause.Child(i)
			if child != nil && (child.Type() == "name" || child.Type() == "qualified_name") {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetSymbol: child.Content(content),
					Kind:         RelExtends,
					Line:         int(child.StartPoint().Row) + 1,
				})
			}
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "class_interface_clause" {
			for j := 0; j < int(child.ChildCount()); j++ {
				iface := child.Child(j)
				if iface != nil && (iface.Type() == "name" || iface.Type() == "qualified_name") {
					analysis.Relationships = append(analysis.Relationships, Relationship{
						TargetSymbol: iface.Content(content),
						Kind:         RelImplements,
						Line:         int(iface.StartPoint().Row) + 1,
					})
				}
			}
		}
	}
}

func (p *PHPParser) parseInterfaceExtends(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	baseClause := node.ChildByFieldName("base_clause")
	if baseClause != nil {
		for i := 0; i < int(baseClause.ChildCount()); i++ {
			child := baseClause.Child(i)
			if child != nil && (child.Type() == "name" || child.Type() == "qualified_name") {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetSymbol: child.Content(content),
					Kind:         RelExtends,
					Line:         int(child.StartPoint().Row) + 1,
				})
			}
		}
	}
}

func (p *PHPParser) isPublic(node *sitter.Node, content []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "visibility_modifier" {
			mod := child.Content(content)
			return mod == "public"
		}
	}
	return true
}

func (p *PHPParser) extractFunctionSignature(node *sitter.Node, content []byte) string {
	nameNode := node.ChildByFieldName("name")
	paramsNode := node.ChildByFieldName("parameters")

	var sig strings.Builder
	sig.WriteString("function ")
	if nameNode != nil {
		sig.WriteString(nameNode.Content(content))
	}
	if paramsNode != nil {
		sig.WriteString(paramsNode.Content(content))
	}

	return sig.String()
}

func (p *PHPParser) extractMethodSignature(node *sitter.Node, content []byte) string {
	nameNode := node.ChildByFieldName("name")
	paramsNode := node.ChildByFieldName("parameters")

	var sig strings.Builder
	if nameNode != nil {
		sig.WriteString(nameNode.Content(content))
	}
	if paramsNode != nil {
		sig.WriteString(paramsNode.Content(content))
	}

	return sig.String()
}

func (p *PHPParser) extractDocComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		comment := prev.Content(content)
		if strings.HasPrefix(comment, "/**") {
			comment = strings.TrimPrefix(comment, "/**")
			comment = strings.TrimSuffix(comment, "*/")
			comment = strings.TrimSpace(comment)
			lines := strings.Split(comment, "\n")
			if len(lines) > 0 {
				first := strings.TrimSpace(lines[0])
				first = strings.TrimPrefix(first, "* ")
				return first
			}
		}
		return strings.TrimPrefix(comment, "// ")
	}

	return ""
}
