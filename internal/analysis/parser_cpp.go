package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
)

type CPPParser struct {
	parser *sitter.Parser
}

func NewCPPParser() *CPPParser {
	p := sitter.NewParser()
	p.SetLanguage(cpp.GetLanguage())
	return &CPPParser{parser: p}
}

func (p *CPPParser) Language() Language {
	return LangCPP
}

func (p *CPPParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangCPP),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *CPPParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_definition":
			sym := p.parseFunctionDef(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "class_specifier":
			sym := p.parseClass(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "struct_specifier":
			sym := p.parseStruct(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "enum_specifier":
			sym := p.parseEnum(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "namespace_definition":
			p.parseNamespace(child, content, analysis)

		case "template_declaration":
			p.parseTemplate(child, content, analysis)

		case "declaration":
			p.parseDeclaration(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *CPPParser) parseFunctionDef(node *sitter.Node, content []byte) *Symbol {
	declarator := node.ChildByFieldName("declarator")
	if declarator == nil {
		return nil
	}

	name := p.extractDeclaratorName(declarator, content)
	if name == "" {
		return nil
	}

	doc := p.extractPrecedingComment(node, content)
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

func (p *CPPParser) parseClass(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractPrecedingComment(node, content)
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

func (p *CPPParser) parseStruct(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body != nil {
		children = p.extractClassMembers(body, content)
	}

	return &Symbol{
		Name:      name,
		Kind:      KindClass,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
		Children:  children,
	}
}

func (p *CPPParser) extractClassMembers(node *sitter.Node, content []byte) []Symbol {
	var members []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "field_declaration":
			declarator := child.ChildByFieldName("declarator")
			if declarator != nil {
				name := p.extractDeclaratorName(declarator, content)
				if name != "" {
					members = append(members, Symbol{
						Name:      name,
						Kind:      KindProperty,
						LineStart: int(child.StartPoint().Row) + 1,
						LineEnd:   int(child.EndPoint().Row) + 1,
						Exported:  true,
					})
				}
			}

		case "function_definition":
			sym := p.parseFunctionDef(child, content)
			if sym != nil {
				sym.Kind = KindMethod
				members = append(members, *sym)
			}

		case "declaration":
			declarator := child.ChildByFieldName("declarator")
			if declarator != nil && declarator.Type() == "function_declarator" {
				name := p.extractDeclaratorName(declarator, content)
				if name != "" {
					members = append(members, Symbol{
						Name:      name,
						Kind:      KindMethod,
						LineStart: int(child.StartPoint().Row) + 1,
						LineEnd:   int(child.EndPoint().Row) + 1,
						Exported:  true,
					})
				}
			}
		}
	}
	return members
}

func (p *CPPParser) parseEnum(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body != nil {
		for i := 0; i < int(body.ChildCount()); i++ {
			child := body.Child(i)
			if child != nil && child.Type() == "enumerator" {
				enumName := child.ChildByFieldName("name")
				if enumName != nil {
					children = append(children, Symbol{
						Name:      enumName.Content(content),
						Kind:      KindConstant,
						LineStart: int(child.StartPoint().Row) + 1,
						LineEnd:   int(child.EndPoint().Row) + 1,
						Exported:  true,
					})
				}
			}
		}
	}

	return &Symbol{
		Name:      name,
		Kind:      KindEnum,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
		Children:  children,
	}
}

func (p *CPPParser) parseNamespace(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	name := nameNode.Content(content)
	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      name,
		Kind:      KindType,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	})
}

func (p *CPPParser) parseTemplate(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_definition":
			sym := p.parseFunctionDef(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}
		case "class_specifier":
			sym := p.parseClass(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}
		}
	}
}

func (p *CPPParser) parseDeclaration(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	declarator := node.ChildByFieldName("declarator")
	if declarator == nil {
		return
	}

	if declarator.Type() == "function_declarator" {
		name := p.extractDeclaratorName(declarator, content)
		if name != "" {
			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:      name,
				Kind:      KindFunction,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Exported:  true,
			})
		}
	}
}

func (p *CPPParser) extractDeclaratorName(node *sitter.Node, content []byte) string {
	switch node.Type() {
	case "identifier", "field_identifier":
		return node.Content(content)
	case "qualified_identifier":
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			return nameNode.Content(content)
		}
	case "pointer_declarator", "reference_declarator", "array_declarator", "function_declarator":
		declarator := node.ChildByFieldName("declarator")
		if declarator != nil {
			return p.extractDeclaratorName(declarator, content)
		}
	case "destructor_name":
		return "~" + p.getChildIdentifier(node, content)
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "identifier" || child.Type() == "field_identifier") {
			return child.Content(content)
		}
	}

	return ""
}

func (p *CPPParser) getChildIdentifier(node *sitter.Node, content []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "identifier" {
			return child.Content(content)
		}
	}
	return ""
}

func (p *CPPParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "preproc_include" {
			pathNode := child.ChildByFieldName("path")
			if pathNode != nil {
				path := pathNode.Content(content)
				path = strings.Trim(path, "\"<>")
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: path,
					Kind:       RelImport,
					Line:       int(child.StartPoint().Row) + 1,
				})
			}
		}

		if child.Type() == "base_class_clause" {
			p.parseBaseClasses(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *CPPParser) parseBaseClasses(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "type_identifier" || child.Type() == "qualified_identifier" {
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetSymbol: child.Content(content),
				Kind:         RelExtends,
				Line:         int(child.StartPoint().Row) + 1,
			})
		}
	}
}

func (p *CPPParser) extractFunctionSignature(node *sitter.Node, content []byte) string {
	typeNode := node.ChildByFieldName("type")
	declarator := node.ChildByFieldName("declarator")

	var sig strings.Builder
	if typeNode != nil {
		sig.WriteString(typeNode.Content(content))
		sig.WriteString(" ")
	}
	if declarator != nil {
		sig.WriteString(declarator.Content(content))
	}

	return sig.String()
}

func (p *CPPParser) extractPrecedingComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		comment := prev.Content(content)
		comment = strings.TrimPrefix(comment, "// ")
		comment = strings.TrimPrefix(comment, "/* ")
		comment = strings.TrimSuffix(comment, " */")
		return comment
	}

	return ""
}
