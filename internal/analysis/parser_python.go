package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

type PythonParser struct {
	parser *sitter.Parser
}

func NewPythonParser() *PythonParser {
	p := sitter.NewParser()
	p.SetLanguage(python.GetLanguage())
	return &PythonParser{parser: p}
}

func (p *PythonParser) Language() Language {
	return LangPython
}

func (p *PythonParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangPython),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis, 0)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *PythonParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis, depth int) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_definition":
			sym := p.parseFunctionDef(child, content, depth)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "class_definition":
			sym := p.parseClassDef(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "decorated_definition":
			p.parseDecoratedDef(child, content, analysis, depth)

		case "assignment":
			if depth == 0 {
				p.parseAssignment(child, content, analysis)
			}
		}

		if child.Type() != "class_definition" && child.Type() != "function_definition" {
			p.extractSymbols(child, content, analysis, depth)
		}
	}
}

func (p *PythonParser) parseFunctionDef(node *sitter.Node, content []byte, depth int) *Symbol {
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

	returnType := ""
	returnNode := node.ChildByFieldName("return_type")
	if returnNode != nil {
		returnType = " -> " + returnNode.Content(content)
	}

	doc := p.extractDocstring(node, content)

	kind := KindFunction
	if depth > 0 || strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__") {
		kind = KindMethod
	}

	return &Symbol{
		Name:       name,
		Kind:       kind,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  "def " + name + params + returnType,
		DocComment: doc,
		Exported:   !strings.HasPrefix(name, "_"),
	}
}

func (p *PythonParser) parseClassDef(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocstring(node, content)

	sym := &Symbol{
		Name:       name,
		Kind:       KindClass,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   !strings.HasPrefix(name, "_"),
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		sym.Children = p.parseClassBody(bodyNode, content)
	}

	return sym
}

func (p *PythonParser) parseClassBody(node *sitter.Node, content []byte) []Symbol {
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_definition":
			sym := p.parseFunctionDef(child, content, 1)
			if sym != nil {
				sym.Kind = KindMethod
				children = append(children, *sym)
			}

		case "decorated_definition":
			for j := 0; j < int(child.ChildCount()); j++ {
				inner := child.Child(j)
				if inner != nil && inner.Type() == "function_definition" {
					sym := p.parseFunctionDef(inner, content, 1)
					if sym != nil {
						sym.Kind = KindMethod
						children = append(children, *sym)
					}
				}
			}

		case "expression_statement":
			for j := 0; j < int(child.ChildCount()); j++ {
				expr := child.Child(j)
				if expr != nil && expr.Type() == "assignment" {
					targets := expr.ChildByFieldName("left")
					if targets != nil {
						for k := 0; k < int(targets.ChildCount()); k++ {
							t := targets.Child(k)
							if t != nil && t.Type() == "identifier" {
								children = append(children, Symbol{
									Name:      t.Content(content),
									Kind:      KindProperty,
									LineStart: int(expr.StartPoint().Row) + 1,
									LineEnd:   int(expr.EndPoint().Row) + 1,
								})
							}
						}
						if targets.Type() == "identifier" {
							children = append(children, Symbol{
								Name:      targets.Content(content),
								Kind:      KindProperty,
								LineStart: int(expr.StartPoint().Row) + 1,
								LineEnd:   int(expr.EndPoint().Row) + 1,
							})
						}
					}
				}
			}
		}
	}

	return children
}

func (p *PythonParser) parseDecoratedDef(node *sitter.Node, content []byte, analysis *FileAnalysis, depth int) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_definition":
			sym := p.parseFunctionDef(child, content, depth)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "class_definition":
			sym := p.parseClassDef(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}
		}
	}
}

func (p *PythonParser) parseAssignment(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return
	}

	if leftNode.Type() == "identifier" {
		name := leftNode.Content(content)
		kind := KindVariable
		if strings.ToUpper(name) == name {
			kind = KindConstant
		}

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      name,
			Kind:      kind,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
			Exported:  !strings.HasPrefix(name, "_"),
		})
	}
}

func (p *PythonParser) extractDocstring(node *sitter.Node, content []byte) string {
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return ""
	}

	if bodyNode.ChildCount() == 0 {
		return ""
	}

	firstChild := bodyNode.Child(0)
	if firstChild == nil {
		return ""
	}

	if firstChild.Type() == "expression_statement" {
		if firstChild.ChildCount() > 0 {
			expr := firstChild.Child(0)
			if expr != nil && expr.Type() == "string" {
				doc := expr.Content(content)
				doc = strings.TrimPrefix(doc, "\"\"\"")
				doc = strings.TrimPrefix(doc, "'''")
				doc = strings.TrimSuffix(doc, "\"\"\"")
				doc = strings.TrimSuffix(doc, "'''")
				return strings.TrimSpace(doc)
			}
		}
	}

	return ""
}

func (p *PythonParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_statement":
			p.parseImportStatement(child, content, analysis)

		case "import_from_statement":
			p.parseImportFromStatement(child, content, analysis)

		case "call":
			p.parseCallExpression(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *PythonParser) parseCallExpression(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}

	var targetSymbol string

	switch funcNode.Type() {
	case "identifier":
		// Simple function call: foo()
		targetSymbol = funcNode.Content(content)

	case "attribute":
		// Method call: obj.method() or module.func()
		objectNode := funcNode.ChildByFieldName("object")
		attrNode := funcNode.ChildByFieldName("attribute")
		if objectNode != nil && attrNode != nil {
			targetSymbol = objectNode.Content(content) + "." + attrNode.Content(content)
		} else if attrNode != nil {
			targetSymbol = attrNode.Content(content)
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

func (p *PythonParser) parseImportStatement(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "dotted_name" || child.Type() == "aliased_import" {
			nameNode := child
			if child.Type() == "aliased_import" {
				nameNode = child.ChildByFieldName("name")
			}

			if nameNode != nil {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: nameNode.Content(content),
					Kind:       RelImport,
					Line:       int(node.StartPoint().Row) + 1,
				})
			}
		}
	}
}

func (p *PythonParser) parseImportFromStatement(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	moduleNode := node.ChildByFieldName("module_name")
	if moduleNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			c := node.Child(i)
			if c != nil && (c.Type() == "dotted_name" || c.Type() == "relative_import") {
				moduleNode = c
				break
			}
		}
	}

	if moduleNode != nil {
		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetFile: moduleNode.Content(content),
			Kind:       RelImport,
			Line:       int(node.StartPoint().Row) + 1,
		})
	}
}
