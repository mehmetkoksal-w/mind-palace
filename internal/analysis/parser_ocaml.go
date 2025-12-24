package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/ocaml"
)

type OCamlParser struct {
	parser *sitter.Parser
}

func NewOCamlParser() *OCamlParser {
	p := sitter.NewParser()
	p.SetLanguage(ocaml.GetLanguage())
	return &OCamlParser{parser: p}
}

func (p *OCamlParser) Language() Language {
	return LangOCaml
}

func (p *OCamlParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangOCaml),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *OCamlParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "value_definition":
			p.parseValueDef(child, content, analysis)

		case "let_binding":
			p.parseLetBinding(child, content, analysis)

		case "type_definition":
			p.parseTypeDef(child, content, analysis)

		case "module_definition":
			sym := p.parseModuleDef(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "module_type_definition":
			sym := p.parseModuleTypeDef(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "exception_definition":
			sym := p.parseExceptionDef(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "class_definition":
			sym := p.parseClassDef(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *OCamlParser) parseValueDef(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "let_binding" {
			p.parseLetBinding(child, content, analysis)
		}
	}
}

func (p *OCamlParser) parseLetBinding(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var name string
	isFunction := false

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "value_name":
			name = child.Content(content)
		case "parameter":
			isFunction = true
		case "function_expression", "fun_expression":
			isFunction = true
		}
	}

	if name == "" {
		return
	}

	kind := KindVariable
	if isFunction {
		kind = KindFunction
	}

	doc := p.extractOcamldoc(node, content)
	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:       name,
		Kind:       kind,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
	})
}

func (p *OCamlParser) parseTypeDef(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "type_binding" {
			var name string
			var children []Symbol

			for j := 0; j < int(child.ChildCount()); j++ {
				typeChild := child.Child(j)
				if typeChild == nil {
					continue
				}

				switch typeChild.Type() {
				case "type_constructor":
					name = typeChild.Content(content)
				case "variant_declaration":
					for k := 0; k < int(typeChild.ChildCount()); k++ {
						variant := typeChild.Child(k)
						if variant != nil && variant.Type() == "constructor_declaration" {
							for l := 0; l < int(variant.ChildCount()); l++ {
								varChild := variant.Child(l)
								if varChild != nil && varChild.Type() == "constructor_name" {
									children = append(children, Symbol{
										Name:      varChild.Content(content),
										Kind:      KindConstant,
										LineStart: int(variant.StartPoint().Row) + 1,
										LineEnd:   int(variant.EndPoint().Row) + 1,
										Exported:  true,
									})
								}
							}
						}
					}
				case "record_declaration":
					for k := 0; k < int(typeChild.ChildCount()); k++ {
						field := typeChild.Child(k)
						if field != nil && field.Type() == "field_declaration" {
							for l := 0; l < int(field.ChildCount()); l++ {
								fieldChild := field.Child(l)
								if fieldChild != nil && fieldChild.Type() == "field_name" {
									children = append(children, Symbol{
										Name:      fieldChild.Content(content),
										Kind:      KindProperty,
										LineStart: int(field.StartPoint().Row) + 1,
										LineEnd:   int(field.EndPoint().Row) + 1,
										Exported:  true,
									})
								}
							}
						}
					}
				}
			}

			if name != "" {
				doc := p.extractOcamldoc(child, content)
				analysis.Symbols = append(analysis.Symbols, Symbol{
					Name:       name,
					Kind:       KindType,
					LineStart:  int(child.StartPoint().Row) + 1,
					LineEnd:    int(child.EndPoint().Row) + 1,
					DocComment: doc,
					Exported:   true,
					Children:   children,
				})
			}
		}
	}
}

func (p *OCamlParser) parseModuleDef(node *sitter.Node, content []byte) *Symbol {
	var name string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "module_name" {
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return nil
	}

	doc := p.extractOcamldoc(node, content)
	return &Symbol{
		Name:       name,
		Kind:       KindClass,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
	}
}

func (p *OCamlParser) parseModuleTypeDef(node *sitter.Node, content []byte) *Symbol {
	var name string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "module_type_name" {
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return nil
	}

	doc := p.extractOcamldoc(node, content)
	return &Symbol{
		Name:       name,
		Kind:       KindInterface,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
	}
}

func (p *OCamlParser) parseExceptionDef(node *sitter.Node, content []byte) *Symbol {
	var name string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "constructor_name" {
			name = child.Content(content)
			break
		}
	}

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

func (p *OCamlParser) parseClassDef(node *sitter.Node, content []byte) *Symbol {
	var name string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "class_binding" {
			for j := 0; j < int(child.ChildCount()); j++ {
				bindChild := child.Child(j)
				if bindChild != nil && bindChild.Type() == "class_name" {
					name = bindChild.Content(content)
					break
				}
			}
		}
	}

	if name == "" {
		return nil
	}

	doc := p.extractOcamldoc(node, content)
	return &Symbol{
		Name:       name,
		Kind:       KindClass,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
	}
}

func (p *OCamlParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "open_module" {
			p.parseOpenModule(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *OCamlParser) parseOpenModule(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "module_path" || child.Type() == "module_name" || child.Type() == "extended_module_path") {
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: child.Content(content),
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
			break
		}
	}
}

func (p *OCamlParser) extractOcamldoc(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		comment := prev.Content(content)
		if strings.HasPrefix(comment, "(**") {
			comment = strings.TrimPrefix(comment, "(**")
			comment = strings.TrimSuffix(comment, "*)")
			return strings.TrimSpace(comment)
		}
	}

	return ""
}
