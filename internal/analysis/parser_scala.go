package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/scala"
)

type ScalaParser struct {
	parser *sitter.Parser
}

func NewScalaParser() *ScalaParser {
	p := sitter.NewParser()
	p.SetLanguage(scala.GetLanguage())
	return &ScalaParser{parser: p}
}

func (p *ScalaParser) Language() Language {
	return LangScala
}

func (p *ScalaParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangScala),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *ScalaParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

		case "object_definition":
			sym := p.parseObject(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "trait_definition":
			sym := p.parseTrait(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "function_definition":
			sym := p.parseFunction(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "val_definition", "var_definition":
			p.parseValVar(child, content, analysis)

		case "type_definition":
			p.parseTypeDef(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *ScalaParser) parseClass(node *sitter.Node, content []byte) *Symbol {
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
	doc := p.extractScaladoc(node, content)
	exported := p.isPublic(node, content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "template_body" {
				body = child
				break
			}
		}
	}

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

func (p *ScalaParser) parseObject(node *sitter.Node, content []byte) *Symbol {
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
	doc := p.extractScaladoc(node, content)
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

func (p *ScalaParser) parseTrait(node *sitter.Node, content []byte) *Symbol {
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
	doc := p.extractScaladoc(node, content)
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

func (p *ScalaParser) parseFunction(node *sitter.Node, content []byte) *Symbol {
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
	doc := p.extractScaladoc(node, content)
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

func (p *ScalaParser) parseValVar(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	patternNode := node.ChildByFieldName("pattern")
	if patternNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "identifier" {
				patternNode = child
				break
			}
		}
	}

	if patternNode == nil {
		return
	}

	name := patternNode.Content(content)
	kind := KindVariable
	if node.Type() == "val_definition" {
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

func (p *ScalaParser) parseTypeDef(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "type_identifier" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      nameNode.Content(content),
		Kind:      KindType,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  p.isPublic(node, content),
	})
}

func (p *ScalaParser) extractClassMembers(node *sitter.Node, content []byte) []Symbol {
	var members []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_definition":
			sym := p.parseFunction(child, content)
			if sym != nil {
				sym.Kind = KindMethod
				members = append(members, *sym)
			}

		case "val_definition", "var_definition":
			patternNode := child.ChildByFieldName("pattern")
			if patternNode == nil {
				for j := 0; j < int(child.ChildCount()); j++ {
					c := child.Child(j)
					if c != nil && c.Type() == "identifier" {
						patternNode = c
						break
					}
				}
			}

			if patternNode != nil {
				members = append(members, Symbol{
					Name:      patternNode.Content(content),
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

func (p *ScalaParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_declaration":
			p.parseImport(child, content, analysis)

		case "class_definition", "object_definition", "trait_definition":
			p.parseInheritance(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *ScalaParser) parseImport(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "stable_identifier" || child.Type() == "identifier") {
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: child.Content(content),
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
			break
		}
	}
}

func (p *ScalaParser) parseInheritance(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "extends_clause" {
			for j := 0; j < int(child.ChildCount()); j++ {
				typeNode := child.Child(j)
				if typeNode != nil && (typeNode.Type() == "type_identifier" || typeNode.Type() == "generic_type") {
					analysis.Relationships = append(analysis.Relationships, Relationship{
						TargetSymbol: typeNode.Content(content),
						Kind:         RelExtends,
						Line:         int(typeNode.StartPoint().Row) + 1,
					})
				}
			}
		}
	}
}

func (p *ScalaParser) isPublic(node *sitter.Node, content []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "modifiers" || child.Type() == "access_modifier" {
			mod := child.Content(content)
			if strings.Contains(mod, "private") || strings.Contains(mod, "protected") {
				return false
			}
		}
	}
	return true
}

func (p *ScalaParser) extractFunctionSignature(node *sitter.Node, content []byte) string {
	var sig strings.Builder
	sig.WriteString("def ")

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "identifier":
			sig.WriteString(child.Content(content))
		case "parameters":
			sig.WriteString(child.Content(content))
		case "type":
			sig.WriteString(": ")
			sig.WriteString(child.Content(content))
		}
	}

	return sig.String()
}

func (p *ScalaParser) extractScaladoc(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" || prev.Type() == "block_comment" {
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
