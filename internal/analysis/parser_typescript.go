package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

type TypeScriptParser struct {
	parser *sitter.Parser
}

func NewTypeScriptParser() *TypeScriptParser {
	p := sitter.NewParser()
	p.SetLanguage(typescript.GetLanguage())
	return &TypeScriptParser{parser: p}
}

func (p *TypeScriptParser) Language() Language {
	return LangTypeScript
}

func (p *TypeScriptParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangTypeScript),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *TypeScriptParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_declaration":
			sym := p.parseFunctionDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

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

		case "type_alias_declaration":
			sym := p.parseTypeAlias(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "enum_declaration":
			sym := p.parseEnumDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "lexical_declaration", "variable_declaration":
			p.parseVariableDecl(child, content, analysis)

		case "export_statement":
			p.parseExportStatement(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *TypeScriptParser) parseFunctionDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	sig := p.buildSignature(node, content)

	return &Symbol{
		Name:      name,
		Kind:      KindFunction,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: sig,
	}
}

func (p *TypeScriptParser) parseClassDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	sym := &Symbol{
		Name:      name,
		Kind:      KindClass,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		sym.Children = p.parseClassBody(bodyNode, content)
	}

	return sym
}

func (p *TypeScriptParser) parseInterfaceDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	sym := &Symbol{
		Name:      name,
		Kind:      KindInterface,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		sym.Children = p.parseInterfaceBody(bodyNode, content)
	}

	return sym
}

func (p *TypeScriptParser) parseTypeAlias(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	return &Symbol{
		Name:      nameNode.Content(content),
		Kind:      KindType,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
	}
}

func (p *TypeScriptParser) parseEnumDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	return &Symbol{
		Name:      nameNode.Content(content),
		Kind:      KindEnum,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
	}
}

func (p *TypeScriptParser) parseClassBody(node *sitter.Node, content []byte) []Symbol {
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "method_definition", "method_signature":
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			name := nameNode.Content(content)
			kind := KindMethod
			if name == "constructor" {
				kind = KindConstructor
			}

			children = append(children, Symbol{
				Name:      name,
				Kind:      kind,
				LineStart: int(child.StartPoint().Row) + 1,
				LineEnd:   int(child.EndPoint().Row) + 1,
			})

		case "public_field_definition", "property_signature":
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			children = append(children, Symbol{
				Name:      nameNode.Content(content),
				Kind:      KindProperty,
				LineStart: int(child.StartPoint().Row) + 1,
				LineEnd:   int(child.EndPoint().Row) + 1,
			})
		}
	}

	return children
}

func (p *TypeScriptParser) parseInterfaceBody(node *sitter.Node, content []byte) []Symbol {
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "method_signature":
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			children = append(children, Symbol{
				Name:      nameNode.Content(content),
				Kind:      KindMethod,
				LineStart: int(child.StartPoint().Row) + 1,
				LineEnd:   int(child.EndPoint().Row) + 1,
			})

		case "property_signature":
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			children = append(children, Symbol{
				Name:      nameNode.Content(content),
				Kind:      KindProperty,
				LineStart: int(child.StartPoint().Row) + 1,
				LineEnd:   int(child.EndPoint().Row) + 1,
			})
		}
	}

	return children
}

func (p *TypeScriptParser) parseVariableDecl(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	isConst := false
	for i := 0; i < int(node.ChildCount()); i++ {
		c := node.Child(i)
		if c != nil && c.Content(content) == "const" {
			isConst = true
			break
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil || child.Type() != "variable_declarator" {
			continue
		}

		nameNode := child.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}

		kind := KindVariable
		if isConst {
			kind = KindConstant
		}

		valueNode := child.ChildByFieldName("value")
		if valueNode != nil {
			switch valueNode.Type() {
			case "arrow_function", "function":
				kind = KindFunction
			case "class":
				kind = KindClass
			}
		}

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      nameNode.Content(content),
			Kind:      kind,
			LineStart: int(child.StartPoint().Row) + 1,
			LineEnd:   int(child.EndPoint().Row) + 1,
		})
	}
}

func (p *TypeScriptParser) parseExportStatement(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_declaration":
			sym := p.parseFunctionDecl(child, content)
			if sym != nil {
				sym.Exported = true
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "class_declaration":
			sym := p.parseClassDecl(child, content)
			if sym != nil {
				sym.Exported = true
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "interface_declaration":
			sym := p.parseInterfaceDecl(child, content)
			if sym != nil {
				sym.Exported = true
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "type_alias_declaration":
			sym := p.parseTypeAlias(child, content)
			if sym != nil {
				sym.Exported = true
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "enum_declaration":
			sym := p.parseEnumDecl(child, content)
			if sym != nil {
				sym.Exported = true
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "lexical_declaration":
			startLen := len(analysis.Symbols)
			p.parseVariableDecl(child, content, analysis)
			for j := startLen; j < len(analysis.Symbols); j++ {
				analysis.Symbols[j].Exported = true
			}
		}
	}
}

func (p *TypeScriptParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_statement":
			p.parseImport(child, content, analysis)

		case "call_expression":
			p.parseCallExpression(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *TypeScriptParser) parseCallExpression(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}

	var targetSymbol string

	switch funcNode.Type() {
	case "identifier":
		// Simple function call: foo()
		targetSymbol = funcNode.Content(content)

	case "member_expression":
		// Method call: obj.method() or chained calls
		objectNode := funcNode.ChildByFieldName("object")
		propertyNode := funcNode.ChildByFieldName("property")
		if objectNode != nil && propertyNode != nil {
			targetSymbol = objectNode.Content(content) + "." + propertyNode.Content(content)
		} else if propertyNode != nil {
			targetSymbol = propertyNode.Content(content)
		}

	case "super":
		targetSymbol = "super"

	case "new_expression":
		// Skip new expressions, they're handled separately
		return
	}

	if targetSymbol != "" {
		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetSymbol: targetSymbol,
			Kind:         RelCall,
			Line:         int(node.StartPoint().Row) + 1,
		})
	}
}

func (p *TypeScriptParser) parseImport(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	sourceNode := node.ChildByFieldName("source")
	if sourceNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			c := node.Child(i)
			if c != nil && c.Type() == "string" {
				sourceNode = c
				break
			}
		}
	}

	if sourceNode != nil {
		importPath := strings.Trim(sourceNode.Content(content), "\"'`")
		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetFile: importPath,
			Kind:       RelImport,
			Line:       int(node.StartPoint().Row) + 1,
		})
	}
}

func (p *TypeScriptParser) buildSignature(node *sitter.Node, content []byte) string {
	var sig strings.Builder

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		sig.WriteString(nameNode.Content(content))
	}

	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		sig.WriteString(paramsNode.Content(content))
	}

	returnNode := node.ChildByFieldName("return_type")
	if returnNode != nil {
		sig.WriteString(": ")
		sig.WriteString(returnNode.Content(content))
	}

	return sig.String()
}
