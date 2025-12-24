package analysis

import (
	"context"
	"strings"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/csharp"
)

type CSharpParser struct {
	parser *sitter.Parser
}

func NewCSharpParser() *CSharpParser {
	p := sitter.NewParser()
	p.SetLanguage(csharp.GetLanguage())
	return &CSharpParser{parser: p}
}

func (p *CSharpParser) Language() Language {
	return LangCSharp
}

func (p *CSharpParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangCSharp),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *CSharpParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

		case "struct_declaration":
			sym := p.parseStruct(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "enum_declaration":
			sym := p.parseEnum(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "method_declaration":
			sym := p.parseMethod(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "namespace_declaration":
			p.parseNamespace(child, content, analysis)

		case "field_declaration", "property_declaration":
			p.parseFieldOrProperty(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *CSharpParser) parseClass(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractXmlDoc(node, content)
	exported := p.isPublic(node, content)
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
		Exported:   exported,
		Children:   children,
	}
}

func (p *CSharpParser) parseInterface(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractXmlDoc(node, content)
	exported := p.isPublic(node, content)
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
		Exported:   exported,
		Children:   children,
	}
}

func (p *CSharpParser) parseStruct(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractXmlDoc(node, content)
	exported := p.isPublic(node, content)
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
		Exported:   exported,
		Children:   children,
	}
}

func (p *CSharpParser) parseEnum(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractXmlDoc(node, content)
	exported := p.isPublic(node, content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body != nil {
		for i := 0; i < int(body.ChildCount()); i++ {
			child := body.Child(i)
			if child != nil && child.Type() == "enum_member_declaration" {
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
		Name:       name,
		Kind:       KindEnum,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   exported,
		Children:   children,
	}
}

func (p *CSharpParser) parseMethod(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractXmlDoc(node, content)
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

func (p *CSharpParser) parseNamespace(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

func (p *CSharpParser) parseFieldOrProperty(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var nameNode *sitter.Node

	if node.Type() == "property_declaration" {
		nameNode = node.ChildByFieldName("name")
	} else {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "variable_declaration" {
				for j := 0; j < int(child.ChildCount()); j++ {
					declarator := child.Child(j)
					if declarator != nil && declarator.Type() == "variable_declarator" {
						nameNode = declarator.ChildByFieldName("name")
						break
					}
				}
			}
		}
	}

	if nameNode == nil {
		return
	}

	name := nameNode.Content(content)
	exported := p.isPublic(node, content)

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      name,
		Kind:      KindProperty,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  exported,
	})
}

func (p *CSharpParser) extractClassMembers(node *sitter.Node, content []byte) []Symbol {
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

		case "constructor_declaration":
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				members = append(members, Symbol{
					Name:      nameNode.Content(content),
					Kind:      KindConstructor,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  p.isPublic(child, content),
				})
			}

		case "property_declaration":
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				members = append(members, Symbol{
					Name:      nameNode.Content(content),
					Kind:      KindProperty,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  p.isPublic(child, content),
				})
			}

		case "field_declaration":
			for j := 0; j < int(child.ChildCount()); j++ {
				varDecl := child.Child(j)
				if varDecl != nil && varDecl.Type() == "variable_declaration" {
					for k := 0; k < int(varDecl.ChildCount()); k++ {
						declarator := varDecl.Child(k)
						if declarator != nil && declarator.Type() == "variable_declarator" {
							nameNode := declarator.ChildByFieldName("name")
							if nameNode != nil {
								members = append(members, Symbol{
									Name:      nameNode.Content(content),
									Kind:      KindProperty,
									LineStart: int(child.StartPoint().Row) + 1,
									LineEnd:   int(child.EndPoint().Row) + 1,
									Exported:  p.isPublic(child, content),
								})
							}
						}
					}
				}
			}
		}
	}
	return members
}

func (p *CSharpParser) extractInterfaceMembers(node *sitter.Node, content []byte) []Symbol {
	var members []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "method_declaration":
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

		case "property_declaration":
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				members = append(members, Symbol{
					Name:      nameNode.Content(content),
					Kind:      KindProperty,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}
	return members
}

func (p *CSharpParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "using_directive":
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: nameNode.Content(content),
					Kind:       RelImport,
					Line:       int(child.StartPoint().Row) + 1,
				})
			}

		case "base_list":
			p.parseBaseList(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *CSharpParser) parseBaseList(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		typeName := ""
		switch child.Type() {
		case "identifier", "generic_name", "qualified_name":
			typeName = child.Content(content)
		}

		if typeName != "" {
			kind := RelExtends
			if strings.HasPrefix(typeName, "I") && len(typeName) > 1 && unicode.IsUpper(rune(typeName[1])) {
				kind = RelImplements
			}

			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetSymbol: typeName,
				Kind:         kind,
				Line:         int(child.StartPoint().Row) + 1,
			})
		}
	}
}

func (p *CSharpParser) isPublic(node *sitter.Node, content []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "modifier" {
			mod := child.Content(content)
			if mod == "public" || mod == "internal" {
				return true
			}
			if mod == "private" || mod == "protected" {
				return false
			}
		}
	}
	return false
}

func (p *CSharpParser) extractMethodSignature(node *sitter.Node, content []byte) string {
	typeNode := node.ChildByFieldName("type")
	nameNode := node.ChildByFieldName("name")
	paramsNode := node.ChildByFieldName("parameters")

	var sig strings.Builder
	if typeNode != nil {
		sig.WriteString(typeNode.Content(content))
		sig.WriteString(" ")
	}
	if nameNode != nil {
		sig.WriteString(nameNode.Content(content))
	}
	if paramsNode != nil {
		sig.WriteString(paramsNode.Content(content))
	}

	return sig.String()
}

func (p *CSharpParser) extractXmlDoc(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		comment := prev.Content(content)
		if strings.HasPrefix(comment, "///") {
			return strings.TrimPrefix(comment, "/// ")
		}
		comment = strings.TrimPrefix(comment, "// ")
		return comment
	}

	return ""
}
