package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/groovy"
)

type GroovyParser struct {
	parser *sitter.Parser
}

func NewGroovyParser() *GroovyParser {
	p := sitter.NewParser()
	p.SetLanguage(groovy.GetLanguage())
	return &GroovyParser{parser: p}
}

func (p *GroovyParser) Language() Language {
	return LangGroovy
}

func (p *GroovyParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangGroovy),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *GroovyParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "class_definition":
			sym := p.parseClass(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "interface_definition":
			sym := p.parseInterface(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "trait_definition":
			sym := p.parseTrait(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "enum_definition":
			sym := p.parseEnum(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "method_definition", "function_definition":
			sym := p.parseMethod(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "variable_definition":
			p.parseVariable(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *GroovyParser) parseClass(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "identifier" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractGroovydoc(node, content)
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

func (p *GroovyParser) parseInterface(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "identifier" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractGroovydoc(node, content)
	exported := p.isPublic(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindInterface,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   exported,
	}
}

func (p *GroovyParser) parseTrait(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "identifier" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractGroovydoc(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindInterface,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
	}
}

func (p *GroovyParser) parseEnum(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "identifier" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractGroovydoc(node, content)
	exported := p.isPublic(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindEnum,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   exported,
	}
}

func (p *GroovyParser) parseMethod(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "identifier" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractGroovydoc(node, content)
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

func (p *GroovyParser) parseVariable(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "identifier" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return
	}

	name := nameNode.Content(content)
	kind := KindVariable

	if strings.ToUpper(name) == name {
		kind = KindConstant
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      name,
		Kind:      kind,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  p.isPublic(node, content),
	})
}

func (p *GroovyParser) extractClassMembers(node *sitter.Node, content []byte) []Symbol {
	var members []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "method_definition", "function_definition":
			sym := p.parseMethod(child, content)
			if sym != nil {
				members = append(members, *sym)
			}

		case "field_definition", "variable_definition":
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				for j := 0; j < int(child.ChildCount()); j++ {
					c := child.Child(j)
					if c != nil && c.Type() == "identifier" {
						nameNode = c
						break
					}
				}
			}

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
	return members
}

func (p *GroovyParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_declaration":
			p.parseImport(child, content, analysis)

		case "class_definition":
			p.parseInheritance(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *GroovyParser) parseImport(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "dotted_identifier" || child.Type() == "identifier") {
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: child.Content(content),
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
			break
		}
	}
}

func (p *GroovyParser) parseInheritance(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "superclass" {
			for j := 0; j < int(child.ChildCount()); j++ {
				typeNode := child.Child(j)
				if typeNode != nil && (typeNode.Type() == "identifier" || typeNode.Type() == "dotted_identifier") {
					analysis.Relationships = append(analysis.Relationships, Relationship{
						TargetSymbol: typeNode.Content(content),
						Kind:         RelExtends,
						Line:         int(typeNode.StartPoint().Row) + 1,
					})
				}
			}
		}

		if child.Type() == "interfaces" {
			for j := 0; j < int(child.ChildCount()); j++ {
				typeNode := child.Child(j)
				if typeNode != nil && (typeNode.Type() == "identifier" || typeNode.Type() == "dotted_identifier") {
					analysis.Relationships = append(analysis.Relationships, Relationship{
						TargetSymbol: typeNode.Content(content),
						Kind:         RelImplements,
						Line:         int(typeNode.StartPoint().Row) + 1,
					})
				}
			}
		}
	}
}

func (p *GroovyParser) isPublic(node *sitter.Node, content []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "modifiers" || child.Type() == "modifier" {
			mod := child.Content(content)
			if strings.Contains(mod, "private") {
				return false
			}
			if strings.Contains(mod, "public") {
				return true
			}
		}
	}
	return true
}

func (p *GroovyParser) extractMethodSignature(node *sitter.Node, content []byte) string {
	var sig strings.Builder

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "identifier":
			sig.WriteString(child.Content(content))
		case "parameters", "formal_parameters":
			sig.WriteString(child.Content(content))
		}
	}

	return sig.String()
}

func (p *GroovyParser) extractGroovydoc(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" || prev.Type() == "groovydoc_comment" {
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
