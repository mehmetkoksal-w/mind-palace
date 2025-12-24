package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/kotlin"
)

type KotlinParser struct {
	parser *sitter.Parser
}

func NewKotlinParser() *KotlinParser {
	p := sitter.NewParser()
	p.SetLanguage(kotlin.GetLanguage())
	return &KotlinParser{parser: p}
}

func (p *KotlinParser) Language() Language {
	return LangKotlin
}

func (p *KotlinParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangKotlin),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *KotlinParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

		case "object_declaration":
			sym := p.parseObject(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "function_declaration":
			sym := p.parseFunction(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "property_declaration":
			p.parseProperty(child, content, analysis)

		case "type_alias":
			p.parseTypeAlias(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *KotlinParser) parseClass(node *sitter.Node, content []byte) *Symbol {
	var name string
	var nameNode *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "type_identifier" {
			nameNode = child
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return nil
	}

	doc := p.extractKDoc(node, content)
	exported := p.isPublic(node, content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "class_body" {
				body = child
				break
			}
		}
	}

	if body != nil {
		children = p.extractClassMembers(body, content)
	}

	kind := KindClass
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Content(content) == "interface" {
			kind = KindInterface
			break
		}
	}

	return &Symbol{
		Name:       name,
		Kind:       kind,
		LineStart:  int(nameNode.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   exported,
		Children:   children,
	}
}

func (p *KotlinParser) parseObject(node *sitter.Node, content []byte) *Symbol {
	var name string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "type_identifier" {
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return nil
	}

	doc := p.extractKDoc(node, content)
	exported := p.isPublic(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindClass,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   exported,
	}
}

func (p *KotlinParser) parseFunction(node *sitter.Node, content []byte) *Symbol {
	var name string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "simple_identifier" {
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return nil
	}

	doc := p.extractKDoc(node, content)
	sig := p.extractFunctionSignature(node, content)
	exported := p.isPublic(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindFunction,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  sig,
		DocComment: doc,
		Exported:   exported,
	}
}

func (p *KotlinParser) parseProperty(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var name string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "variable_declaration" {
			for j := 0; j < int(child.ChildCount()); j++ {
				id := child.Child(j)
				if id != nil && id.Type() == "simple_identifier" {
					name = id.Content(content)
					break
				}
			}
			break
		}
	}

	if name == "" {
		return
	}

	kind := KindVariable
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Content(content) == "val" {
			kind = KindConstant
			break
		}
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      name,
		Kind:      kind,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  p.isPublic(node, content),
	})
}

func (p *KotlinParser) parseTypeAlias(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var name string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "type_identifier" {
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      name,
		Kind:      KindType,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  p.isPublic(node, content),
	})
}

func (p *KotlinParser) extractClassMembers(node *sitter.Node, content []byte) []Symbol {
	var members []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_declaration":
			sym := p.parseFunction(child, content)
			if sym != nil {
				sym.Kind = KindMethod
				members = append(members, *sym)
			}

		case "property_declaration":
			var name string
			for j := 0; j < int(child.ChildCount()); j++ {
				varDecl := child.Child(j)
				if varDecl != nil && varDecl.Type() == "variable_declaration" {
					for k := 0; k < int(varDecl.ChildCount()); k++ {
						id := varDecl.Child(k)
						if id != nil && id.Type() == "simple_identifier" {
							name = id.Content(content)
							break
						}
					}
					break
				}
			}

			if name != "" {
				members = append(members, Symbol{
					Name:      name,
					Kind:      KindProperty,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  p.isPublic(child, content),
				})
			}
		}
	}
	return members
}

func (p *KotlinParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_header":
			p.parseImport(child, content, analysis)

		case "class_declaration":
			p.parseInheritance(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *KotlinParser) parseImport(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "identifier" {
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: child.Content(content),
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
			break
		}
	}
}

func (p *KotlinParser) parseInheritance(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "delegation_specifiers" {
			for j := 0; j < int(child.ChildCount()); j++ {
				spec := child.Child(j)
				if spec != nil {
					var typeName string
					for k := 0; k < int(spec.ChildCount()); k++ {
						typeNode := spec.Child(k)
						if typeNode != nil && (typeNode.Type() == "user_type" || typeNode.Type() == "type_identifier") {
							typeName = typeNode.Content(content)
							break
						}
					}

					if typeName != "" {
						kind := RelExtends
						analysis.Relationships = append(analysis.Relationships, Relationship{
							TargetSymbol: typeName,
							Kind:         kind,
							Line:         int(spec.StartPoint().Row) + 1,
						})
					}
				}
			}
		}
	}
}

func (p *KotlinParser) isPublic(node *sitter.Node, content []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "modifiers" || child.Type() == "modifier" {
			mod := child.Content(content)
			if strings.Contains(mod, "private") || strings.Contains(mod, "internal") {
				return false
			}
		}
	}
	return true
}

func (p *KotlinParser) extractFunctionSignature(node *sitter.Node, content []byte) string {
	var sig strings.Builder
	sig.WriteString("fun ")

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "simple_identifier":
			sig.WriteString(child.Content(content))
		case "function_value_parameters":
			sig.WriteString(child.Content(content))
		case "type":
			sig.WriteString(": ")
			sig.WriteString(child.Content(content))
		}
	}

	return sig.String()
}

func (p *KotlinParser) extractKDoc(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "multiline_comment" {
		comment := prev.Content(content)
		if strings.HasPrefix(comment, "/**") {
			comment = strings.TrimPrefix(comment, "/**")
			comment = strings.TrimSuffix(comment, "*/")
			lines := strings.Split(comment, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "* ")
				if line != "" && !strings.HasPrefix(line, "@") {
					return line
				}
			}
		}
	}

	return ""
}
