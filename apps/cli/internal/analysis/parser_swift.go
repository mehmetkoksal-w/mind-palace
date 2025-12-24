package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/swift"
)

type SwiftParser struct {
	parser *sitter.Parser
}

func NewSwiftParser() *SwiftParser {
	p := sitter.NewParser()
	p.SetLanguage(swift.GetLanguage())
	return &SwiftParser{parser: p}
}

func (p *SwiftParser) Language() Language {
	return LangSwift
}

func (p *SwiftParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangSwift),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *SwiftParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

		case "struct_declaration":
			sym := p.parseStruct(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "protocol_declaration":
			sym := p.parseProtocol(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "enum_declaration":
			sym := p.parseEnum(child, content)
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

		case "typealias_declaration":
			p.parseTypealias(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *SwiftParser) parseClass(node *sitter.Node, content []byte) *Symbol {
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
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocComment(node, content)
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

func (p *SwiftParser) parseStruct(node *sitter.Node, content []byte) *Symbol {
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
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocComment(node, content)
	exported := p.isPublic(node, content)
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "class_body" || child.Type() == "struct_body") {
			children = p.extractClassMembers(child, content)
			break
		}
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

func (p *SwiftParser) parseProtocol(node *sitter.Node, content []byte) *Symbol {
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
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocComment(node, content)
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

func (p *SwiftParser) parseEnum(node *sitter.Node, content []byte) *Symbol {
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
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocComment(node, content)
	exported := p.isPublic(node, content)
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "enum_class_body" {
			children = p.extractEnumCases(child, content)
			break
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

func (p *SwiftParser) parseFunction(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "simple_identifier" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractDocComment(node, content)
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

func (p *SwiftParser) parseProperty(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var name string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "pattern" {
			for j := 0; j < int(child.ChildCount()); j++ {
				patternChild := child.Child(j)
				if patternChild != nil && patternChild.Type() == "simple_identifier" {
					name = patternChild.Content(content)
					break
				}
			}
			if name != "" {
				break
			}
		}
	}

	if name == "" {
		return
	}

	kind := KindVariable
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Content(content) == "let" {
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

func (p *SwiftParser) parseTypealias(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

func (p *SwiftParser) extractClassMembers(node *sitter.Node, content []byte) []Symbol {
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
				pattern := child.Child(j)
				if pattern != nil && pattern.Type() == "pattern" {
					for k := 0; k < int(pattern.ChildCount()); k++ {
						id := pattern.Child(k)
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

		case "init_declaration":
			members = append(members, Symbol{
				Name:      "init",
				Kind:      KindConstructor,
				LineStart: int(child.StartPoint().Row) + 1,
				LineEnd:   int(child.EndPoint().Row) + 1,
				Exported:  p.isPublic(child, content),
			})
		}
	}
	return members
}

func (p *SwiftParser) extractEnumCases(node *sitter.Node, content []byte) []Symbol {
	var cases []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "enum_entry" {
			for j := 0; j < int(child.ChildCount()); j++ {
				nameNode := child.Child(j)
				if nameNode != nil && nameNode.Type() == "simple_identifier" {
					cases = append(cases, Symbol{
						Name:      nameNode.Content(content),
						Kind:      KindConstant,
						LineStart: int(child.StartPoint().Row) + 1,
						LineEnd:   int(child.EndPoint().Row) + 1,
						Exported:  true,
					})
					break
				}
			}
		}
	}
	return cases
}

func (p *SwiftParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "import_declaration":
			p.parseImport(child, content, analysis)

		case "class_declaration", "struct_declaration":
			p.parseInheritance(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *SwiftParser) parseImport(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

func (p *SwiftParser) parseInheritance(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "inheritance_specifier" || child.Type() == "type_inheritance_clause" {
			for j := 0; j < int(child.ChildCount()); j++ {
				typeNode := child.Child(j)
				if typeNode != nil && typeNode.Type() == "type_identifier" {
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

func (p *SwiftParser) isPublic(node *sitter.Node, content []byte) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "modifiers" {
			mod := child.Content(content)
			if strings.Contains(mod, "private") || strings.Contains(mod, "fileprivate") {
				return false
			}
			if strings.Contains(mod, "public") || strings.Contains(mod, "open") {
				return true
			}
		}
	}
	return true
}

func (p *SwiftParser) extractFunctionSignature(node *sitter.Node, content []byte) string {
	var sig strings.Builder
	sig.WriteString("func ")

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
		case "type_annotation":
			sig.WriteString(" -> ")
			sig.WriteString(child.Content(content))
		}
	}

	return sig.String()
}

func (p *SwiftParser) extractDocComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" || prev.Type() == "multiline_comment" {
		comment := prev.Content(content)
		if strings.HasPrefix(comment, "///") {
			return strings.TrimPrefix(comment, "/// ")
		}
		if strings.HasPrefix(comment, "/**") {
			comment = strings.TrimPrefix(comment, "/**")
			comment = strings.TrimSuffix(comment, "*/")
			lines := strings.Split(comment, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "* ")
				if line != "" && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "@") {
					return line
				}
			}
		}
	}

	return ""
}
