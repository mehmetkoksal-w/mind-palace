package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/elm"
)

type ElmParser struct {
	parser *sitter.Parser
}

func NewElmParser() *ElmParser {
	p := sitter.NewParser()
	p.SetLanguage(elm.GetLanguage())
	return &ElmParser{parser: p}
}

func (p *ElmParser) Language() Language {
	return LangElm
}

func (p *ElmParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangElm),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *ElmParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "module_declaration":
			sym := p.parseModuleDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "type_declaration":
			sym := p.parseTypeDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "type_alias_declaration":
			sym := p.parseTypeAlias(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "value_declaration":
			sym := p.parseValueDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "port_annotation":
			sym := p.parsePort(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *ElmParser) parseModuleDecl(node *sitter.Node, content []byte) *Symbol {
	var name string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "upper_case_qid" || child.Type() == "upper_case_identifier") {
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return nil
	}

	return &Symbol{
		Name:      name,
		Kind:      KindClass,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	}
}

func (p *ElmParser) parseTypeDecl(node *sitter.Node, content []byte) *Symbol {
	var name string
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "upper_case_identifier":
			if name == "" {
				name = child.Content(content)
			}
		case "union_variant":
			variantName := ""
			for j := 0; j < int(child.ChildCount()); j++ {
				varChild := child.Child(j)
				if varChild != nil && varChild.Type() == "upper_case_identifier" {
					variantName = varChild.Content(content)
					break
				}
			}
			if variantName != "" {
				children = append(children, Symbol{
					Name:      variantName,
					Kind:      KindConstant,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}

	if name == "" {
		return nil
	}

	doc := p.extractElmDoc(node, content)
	return &Symbol{
		Name:       name,
		Kind:       KindEnum,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
		Children:   children,
	}
}

func (p *ElmParser) parseTypeAlias(node *sitter.Node, content []byte) *Symbol {
	var name string
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "upper_case_identifier":
			if name == "" {
				name = child.Content(content)
			}
		case "record_type":
			for j := 0; j < int(child.ChildCount()); j++ {
				fieldChild := child.Child(j)
				if fieldChild != nil && fieldChild.Type() == "field_type" {
					for k := 0; k < int(fieldChild.ChildCount()); k++ {
						fieldNameNode := fieldChild.Child(k)
						if fieldNameNode != nil && fieldNameNode.Type() == "lower_case_identifier" {
							children = append(children, Symbol{
								Name:      fieldNameNode.Content(content),
								Kind:      KindProperty,
								LineStart: int(fieldChild.StartPoint().Row) + 1,
								LineEnd:   int(fieldChild.EndPoint().Row) + 1,
								Exported:  true,
							})
							break
						}
					}
				}
			}
		}
	}

	if name == "" {
		return nil
	}

	doc := p.extractElmDoc(node, content)
	return &Symbol{
		Name:       name,
		Kind:       KindType,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
		Children:   children,
	}
}

func (p *ElmParser) parseValueDecl(node *sitter.Node, content []byte) *Symbol {
	var name string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "function_declaration_left" {
			for j := 0; j < int(child.ChildCount()); j++ {
				funcChild := child.Child(j)
				if funcChild != nil && funcChild.Type() == "lower_case_identifier" {
					name = funcChild.Content(content)
					break
				}
			}
		}
	}

	if name == "" {
		return nil
	}

	doc := p.extractElmDoc(node, content)
	return &Symbol{
		Name:       name,
		Kind:       KindFunction,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
	}
}

func (p *ElmParser) parsePort(node *sitter.Node, content []byte) *Symbol {
	var name string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "lower_case_identifier" {
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return nil
	}

	return &Symbol{
		Name:      name,
		Kind:      KindInterface,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: "port",
		Exported:  true,
	}
}

func (p *ElmParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "import_clause" {
			p.parseImport(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *ElmParser) parseImport(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "upper_case_qid" || child.Type() == "upper_case_identifier") {
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: child.Content(content),
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
			break
		}
	}
}

func (p *ElmParser) extractElmDoc(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "block_comment" {
		comment := prev.Content(content)
		if strings.HasPrefix(comment, "{-|") {
			comment = strings.TrimPrefix(comment, "{-|")
			comment = strings.TrimSuffix(comment, "-}")
			lines := strings.Split(comment, "\n")
			if len(lines) > 0 {
				return strings.TrimSpace(lines[0])
			}
		}
	}

	return ""
}
