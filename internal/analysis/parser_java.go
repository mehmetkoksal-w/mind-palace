package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

type JavaParser struct {
	parser *sitter.Parser
}

func NewJavaParser() *JavaParser {
	p := sitter.NewParser()
	p.SetLanguage(java.GetLanguage())
	return &JavaParser{parser: p}
}

func (p *JavaParser) Language() Language {
	return LangJava
}

func (p *JavaParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangJava),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *JavaParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "class_declaration":
			sym := p.parseClassDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "interface_declaration":
			sym := p.parseInterfaceDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "enum_declaration":
			sym := p.parseEnumDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "method_declaration":
			sym := p.parseMethodDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "field_declaration":
			p.parseFieldDecl(child, content, analysis)
		}

		if child.Type() != "class_declaration" && child.Type() != "interface_declaration" {
			p.extractSymbols(child, content, analysis)
		}
	}
}

func (p *JavaParser) parseClassDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	exported := p.isPublic(node, content)
	doc := p.extractJavadoc(node, content)

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
		sym.Children = p.parseClassBody(bodyNode, content)
	}

	return sym
}

func (p *JavaParser) parseInterfaceDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)

	sym := &Symbol{
		Name:       name,
		Kind:       KindInterface,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: p.extractJavadoc(node, content),
		Exported:   p.isPublic(node, content),
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		sym.Children = p.parseInterfaceBody(bodyNode, content)
	}

	return sym
}

func (p *JavaParser) parseEnumDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	return &Symbol{
		Name:       nameNode.Content(content),
		Kind:       KindEnum,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: p.extractJavadoc(node, content),
		Exported:   p.isPublic(node, content),
	}
}

func (p *JavaParser) parseMethodDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)

	returnType := ""
	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		returnType = typeNode.Content(content)
	}

	params := ""
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		params = paramsNode.Content(content)
	}

	return &Symbol{
		Name:       name,
		Kind:       KindMethod,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  returnType + " " + name + params,
		DocComment: p.extractJavadoc(node, content),
		Exported:   p.isPublic(node, content),
	}
}

func (p *JavaParser) parseFieldDecl(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "variable_declarator" {
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			kind := KindProperty
			if p.isFinal(node, content) && p.isStatic(node, content) {
				kind = KindConstant
			}

			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:      nameNode.Content(content),
				Kind:      kind,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Exported:  p.isPublic(node, content),
			})
		}
	}
}

func (p *JavaParser) parseClassBody(node *sitter.Node, content []byte) []Symbol {
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "method_declaration":
			sym := p.parseMethodDecl(child, content)
			if sym != nil {
				children = append(children, *sym)
			}

		case "constructor_declaration":
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				children = append(children, Symbol{
					Name:      nameNode.Content(content),
					Kind:      KindConstructor,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  p.isPublic(child, content),
				})
			}

		case "field_declaration":
			for j := 0; j < int(child.ChildCount()); j++ {
				decl := child.Child(j)
				if decl != nil && decl.Type() == "variable_declarator" {
					nameNode := decl.ChildByFieldName("name")
					if nameNode != nil {
						kind := KindProperty
						if p.isFinal(child, content) && p.isStatic(child, content) {
							kind = KindConstant
						}
						children = append(children, Symbol{
							Name:      nameNode.Content(content),
							Kind:      kind,
							LineStart: int(child.StartPoint().Row) + 1,
							LineEnd:   int(child.EndPoint().Row) + 1,
							Exported:  p.isPublic(child, content),
						})
					}
				}
			}
		}
	}

	return children
}

func (p *JavaParser) parseInterfaceBody(node *sitter.Node, content []byte) []Symbol {
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "method_declaration" {
			sym := p.parseMethodDecl(child, content)
			if sym != nil {
				children = append(children, *sym)
			}
		}
	}

	return children
}

func (p *JavaParser) isPublic(node *sitter.Node, content []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "modifiers" {
			modText := child.Content(content)
			return strings.Contains(modText, "public")
		}
	}
	return false
}

func (p *JavaParser) isFinal(node *sitter.Node, content []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "modifiers" {
			modText := child.Content(content)
			return strings.Contains(modText, "final")
		}
	}
	return false
}

func (p *JavaParser) isStatic(node *sitter.Node, content []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "modifiers" {
			modText := child.Content(content)
			return strings.Contains(modText, "static")
		}
	}
	return false
}

func (p *JavaParser) extractJavadoc(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "block_comment" {
		text := prev.Content(content)
		if strings.HasPrefix(text, "/**") {
			text = strings.TrimPrefix(text, "/**")
			text = strings.TrimSuffix(text, "*/")
			lines := strings.Split(text, "\n")
			var cleaned []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "*")
				line = strings.TrimSpace(line)
				if line != "" {
					cleaned = append(cleaned, line)
				}
			}
			return strings.Join(cleaned, " ")
		}
	}

	return ""
}

func (p *JavaParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_declaration":
			p.parseImport(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *JavaParser) parseImport(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "scoped_identifier" || child.Type() == "identifier" {
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: child.Content(content),
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
			break
		}
	}
}
