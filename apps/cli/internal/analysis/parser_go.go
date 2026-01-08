package analysis

import (
	"context"
	"strings"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

type GoParser struct {
	parser *sitter.Parser
}

func NewGoParser() *GoParser {
	p := sitter.NewParser()
	p.SetLanguage(golang.GetLanguage())
	return &GoParser{parser: p}
}

func (p *GoParser) Language() Language {
	return LangGo
}

func (p *GoParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangGo),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *GoParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

		case "method_declaration":
			sym := p.parseMethodDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "type_declaration":
			p.parseTypeDecl(child, content, analysis)

		case "var_declaration", "const_declaration":
			p.parseVarDecl(child, content, analysis, child.Type() == "const_declaration")
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *GoParser) parseFunctionDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	sig := p.extractSignature(node, content)
	doc := p.extractPrecedingComment(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindFunction,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  sig,
		DocComment: doc,
		Exported:   isExported(name),
	}
}

func (p *GoParser) parseMethodDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	sig := p.extractSignature(node, content)
	doc := p.extractPrecedingComment(node, content)

	receiver := ""
	receiverNode := node.ChildByFieldName("receiver")
	if receiverNode != nil {
		receiver = receiverNode.Content(content)
	}

	fullSig := sig
	if receiver != "" {
		fullSig = receiver + " " + sig
	}

	return &Symbol{
		Name:       name,
		Kind:       KindMethod,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  fullSig,
		DocComment: doc,
		Exported:   isExported(name),
	}
}

func (p *GoParser) parseTypeDecl(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		spec := node.Child(i)
		if spec == nil || spec.Type() != "type_spec" {
			continue
		}

		nameNode := spec.ChildByFieldName("name")
		typeNode := spec.ChildByFieldName("type")
		if nameNode == nil {
			continue
		}

		name := nameNode.Content(content)
		doc := p.extractPrecedingComment(node, content)

		var kind SymbolKind
		var children []Symbol

		if typeNode != nil {
			switch typeNode.Type() {
			case "struct_type":
				kind = KindClass
				children = p.extractStructFields(typeNode, content)
			case "interface_type":
				kind = KindInterface
				children = p.extractInterfaceMethods(typeNode, content)
			default:
				kind = KindType
			}
		} else {
			kind = KindType
		}

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:       name,
			Kind:       kind,
			LineStart:  int(spec.StartPoint().Row) + 1,
			LineEnd:    int(spec.EndPoint().Row) + 1,
			DocComment: doc,
			Exported:   isExported(name),
			Children:   children,
		})
	}
}

func (p *GoParser) extractStructFields(node *sitter.Node, content []byte) []Symbol {
	var fields []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil || child.Type() != "field_declaration_list" {
			continue
		}

		for j := 0; j < int(child.ChildCount()); j++ {
			field := child.Child(j)
			if field == nil || field.Type() != "field_declaration" {
				continue
			}

			nameNode := field.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			fields = append(fields, Symbol{
				Name:      nameNode.Content(content),
				Kind:      KindProperty,
				LineStart: int(field.StartPoint().Row) + 1,
				LineEnd:   int(field.EndPoint().Row) + 1,
				Exported:  isExported(nameNode.Content(content)),
			})
		}
	}
	return fields
}

func (p *GoParser) extractInterfaceMethods(node *sitter.Node, content []byte) []Symbol {
	var methods []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "method_spec" {
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			methods = append(methods, Symbol{
				Name:      nameNode.Content(content),
				Kind:      KindMethod,
				LineStart: int(child.StartPoint().Row) + 1,
				LineEnd:   int(child.EndPoint().Row) + 1,
				Exported:  isExported(nameNode.Content(content)),
			})
		}
	}
	return methods
}

func (p *GoParser) parseVarDecl(node *sitter.Node, content []byte, analysis *FileAnalysis, isConst bool) {
	for i := 0; i < int(node.ChildCount()); i++ {
		spec := node.Child(i)
		if spec == nil || spec.Type() != "var_spec" && spec.Type() != "const_spec" {
			continue
		}

		nameNode := spec.ChildByFieldName("name")
		if nameNode == nil {
			for j := 0; j < int(spec.ChildCount()); j++ {
				c := spec.Child(j)
				if c != nil && c.Type() == "identifier" {
					nameNode = c
					break
				}
			}
		}

		if nameNode == nil {
			continue
		}

		name := nameNode.Content(content)
		kind := KindVariable
		if isConst {
			kind = KindConstant
		}

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      name,
			Kind:      kind,
			LineStart: int(spec.StartPoint().Row) + 1,
			LineEnd:   int(spec.EndPoint().Row) + 1,
			Exported:  isExported(name),
		})
	}
}

func (p *GoParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_declaration":
			p.parseImports(child, content, analysis)

		case "call_expression":
			p.parseCallExpression(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *GoParser) parseCallExpression(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}

	var targetSymbol string

	switch funcNode.Type() {
	case "identifier":
		// Simple function call: foo()
		targetSymbol = funcNode.Content(content)

	case "selector_expression":
		// Method or package call: obj.Method() or pkg.Func()
		operandNode := funcNode.ChildByFieldName("operand")
		fieldNode := funcNode.ChildByFieldName("field")
		if operandNode != nil && fieldNode != nil {
			targetSymbol = operandNode.Content(content) + "." + fieldNode.Content(content)
		} else if fieldNode != nil {
			targetSymbol = fieldNode.Content(content)
		}

	case "parenthesized_expression":
		// Type conversion or complex call: (Type)(value) - skip
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

func (p *GoParser) parseImports(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_spec":
			pathNode := child.ChildByFieldName("path")
			if pathNode == nil {
				for j := 0; j < int(child.ChildCount()); j++ {
					c := child.Child(j)
					if c != nil && c.Type() == "interpreted_string_literal" {
						pathNode = c
						break
					}
				}
			}

			if pathNode != nil {
				importPath := strings.Trim(pathNode.Content(content), "\"")
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: importPath,
					Kind:       RelImport,
					Line:       int(child.StartPoint().Row) + 1,
				})
			}

		case "import_spec_list":
			p.parseImports(child, content, analysis)

		case "interpreted_string_literal":
			importPath := strings.Trim(child.Content(content), "\"")
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: importPath,
				Kind:       RelImport,
				Line:       int(child.StartPoint().Row) + 1,
			})
		}
	}
}

func (p *GoParser) extractSignature(node *sitter.Node, content []byte) string {
	nameNode := node.ChildByFieldName("name")
	paramsNode := node.ChildByFieldName("parameters")
	resultNode := node.ChildByFieldName("result")

	var sig strings.Builder
	if nameNode != nil {
		sig.WriteString(nameNode.Content(content))
	}
	if paramsNode != nil {
		sig.WriteString(paramsNode.Content(content))
	}
	if resultNode != nil {
		sig.WriteString(" ")
		sig.WriteString(resultNode.Content(content))
	}

	return sig.String()
}

func (p *GoParser) extractPrecedingComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		return strings.TrimPrefix(prev.Content(content), "// ")
	}

	return ""
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}
