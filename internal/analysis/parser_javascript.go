package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
)

type JavaScriptParser struct {
	parser *sitter.Parser
}

func NewJavaScriptParser() *JavaScriptParser {
	p := sitter.NewParser()
	p.SetLanguage(javascript.GetLanguage())
	return &JavaScriptParser{parser: p}
}

func (p *JavaScriptParser) Language() Language {
	return LangJavaScript
}

func (p *JavaScriptParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangJavaScript),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *JavaScriptParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

		case "lexical_declaration", "variable_declaration":
			p.parseVariableDecl(child, content, analysis)

		case "export_statement":
			p.parseExportStatement(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *JavaScriptParser) parseFunctionDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	params := ""
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		params = paramsNode.Content(content)
	}

	return &Symbol{
		Name:      name,
		Kind:      KindFunction,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: name + params,
		Exported:  false,
	}
}

func (p *JavaScriptParser) parseClassDecl(node *sitter.Node, content []byte) *Symbol {
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
		Exported:  false,
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		sym.Children = p.parseClassBody(bodyNode, content)
	}

	return sym
}

func (p *JavaScriptParser) parseClassBody(node *sitter.Node, content []byte) []Symbol {
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "method_definition":
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

		case "field_definition":
			nameNode := child.ChildByFieldName("property")
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

func (p *JavaScriptParser) parseVariableDecl(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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
			if valueNode.Type() == "arrow_function" || valueNode.Type() == "function" {
				kind = KindFunction
			} else if valueNode.Type() == "class" {
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

func (p *JavaScriptParser) parseExportStatement(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

		case "lexical_declaration":
			startLen := len(analysis.Symbols)
			p.parseVariableDecl(child, content, analysis)
			for j := startLen; j < len(analysis.Symbols); j++ {
				analysis.Symbols[j].Exported = true
			}
		}
	}
}

func (p *JavaScriptParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_statement":
			p.parseImport(child, content, analysis)

		case "call_expression":
			if p.isRequireCall(child, content) {
				p.parseRequireCall(child, content, analysis)
			} else {
				p.parseCallExpression(child, content, analysis)
			}
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *JavaScriptParser) parseCallExpression(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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
	}

	if targetSymbol != "" {
		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetSymbol: targetSymbol,
			Kind:         RelCall,
			Line:         int(node.StartPoint().Row) + 1,
		})
	}
}

func (p *JavaScriptParser) parseImport(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

func (p *JavaScriptParser) isRequireCall(node *sitter.Node, content []byte) bool {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return false
	}
	return funcNode.Content(content) == "require"
}

func (p *JavaScriptParser) parseRequireCall(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	argsNode := node.ChildByFieldName("arguments")
	if argsNode == nil {
		return
	}

	for i := 0; i < int(argsNode.ChildCount()); i++ {
		arg := argsNode.Child(i)
		if arg != nil && arg.Type() == "string" {
			importPath := strings.Trim(arg.Content(content), "\"'`")
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: importPath,
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
			break
		}
	}
}
