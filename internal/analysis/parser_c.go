package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
)

type CParser struct {
	parser *sitter.Parser
}

func NewCParser() *CParser {
	p := sitter.NewParser()
	p.SetLanguage(c.GetLanguage())
	return &CParser{parser: p}
}

func (p *CParser) Language() Language {
	return LangC
}

func (p *CParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangC),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *CParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

		case "declaration":
			p.parseDeclaration(child, content, analysis)

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

		case "type_definition":
			sym := p.parseTypedef(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *CParser) parseFunctionDef(node *sitter.Node, content []byte) *Symbol {
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

func (p *CParser) parseDeclaration(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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
	} else {
		name := p.extractDeclaratorName(declarator, content)
		if name != "" {
			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:      name,
				Kind:      KindVariable,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Exported:  true,
			})
		}
	}
}

func (p *CParser) parseStruct(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body != nil {
		children = p.extractStructFields(body, content)
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

func (p *CParser) extractStructFields(node *sitter.Node, content []byte) []Symbol {
	var fields []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "field_declaration" {
			declarator := child.ChildByFieldName("declarator")
			if declarator != nil {
				name := p.extractDeclaratorName(declarator, content)
				if name != "" {
					fields = append(fields, Symbol{
						Name:      name,
						Kind:      KindProperty,
						LineStart: int(child.StartPoint().Row) + 1,
						LineEnd:   int(child.EndPoint().Row) + 1,
						Exported:  true,
					})
				}
			}
		}
	}
	return fields
}

func (p *CParser) parseEnum(node *sitter.Node, content []byte) *Symbol {
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

func (p *CParser) parseTypedef(node *sitter.Node, content []byte) *Symbol {
	declarator := node.ChildByFieldName("declarator")
	if declarator == nil {
		return nil
	}

	name := p.extractDeclaratorName(declarator, content)
	if name == "" {
		return nil
	}

	return &Symbol{
		Name:      name,
		Kind:      KindType,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	}
}

func (p *CParser) extractDeclaratorName(node *sitter.Node, content []byte) string {
	switch node.Type() {
	case "identifier":
		return node.Content(content)
	case "pointer_declarator", "array_declarator", "function_declarator":
		declarator := node.ChildByFieldName("declarator")
		if declarator != nil {
			return p.extractDeclaratorName(declarator, content)
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "identifier" {
			return child.Content(content)
		}
	}

	return ""
}

func (p *CParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
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

		p.extractRelationships(child, content, analysis)
	}
}

func (p *CParser) extractFunctionSignature(node *sitter.Node, content []byte) string {
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

func (p *CParser) extractPrecedingComment(node *sitter.Node, content []byte) string {
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
