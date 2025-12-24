package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/protobuf"
)

type ProtobufParser struct {
	parser *sitter.Parser
}

func NewProtobufParser() *ProtobufParser {
	p := sitter.NewParser()
	p.SetLanguage(protobuf.GetLanguage())
	return &ProtobufParser{parser: p}
}

func (p *ProtobufParser) Language() Language {
	return LangProtobuf
}

func (p *ProtobufParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangProtobuf),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *ProtobufParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "message":
			sym := p.parseMessage(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "enum":
			sym := p.parseEnum(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "service":
			sym := p.parseService(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "package":
			sym := p.parsePackage(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "option":
			p.parseOption(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *ProtobufParser) parseMessage(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "message_name" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractComment(node, content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "message_body" {
				body = child
				break
			}
		}
	}

	if body != nil {
		children = p.extractMessageFields(body, content)
	}

	return &Symbol{
		Name:       name,
		Kind:       KindClass,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
		Children:   children,
	}
}

func (p *ProtobufParser) extractMessageFields(node *sitter.Node, content []byte) []Symbol {
	var fields []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "field":
			fieldName := child.ChildByFieldName("name")
			if fieldName == nil {
				for j := 0; j < int(child.ChildCount()); j++ {
					fieldChild := child.Child(j)
					if fieldChild != nil && fieldChild.Type() == "identifier" {
						fieldName = fieldChild
						break
					}
				}
			}

			if fieldName != nil {
				fields = append(fields, Symbol{
					Name:      fieldName.Content(content),
					Kind:      KindProperty,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  true,
				})
			}

		case "message":
			sym := p.parseMessage(child, content)
			if sym != nil {
				fields = append(fields, *sym)
			}

		case "enum":
			sym := p.parseEnum(child, content)
			if sym != nil {
				fields = append(fields, *sym)
			}

		case "oneof":
			oneofName := child.ChildByFieldName("name")
			if oneofName == nil {
				for j := 0; j < int(child.ChildCount()); j++ {
					fieldChild := child.Child(j)
					if fieldChild != nil && fieldChild.Type() == "identifier" {
						oneofName = fieldChild
						break
					}
				}
			}

			if oneofName != nil {
				fields = append(fields, Symbol{
					Name:      oneofName.Content(content),
					Kind:      KindType,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}
	return fields
}

func (p *ProtobufParser) parseEnum(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "enum_name" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractComment(node, content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "enum_body" {
				body = child
				break
			}
		}
	}

	if body != nil {
		for i := 0; i < int(body.ChildCount()); i++ {
			child := body.Child(i)
			if child != nil && child.Type() == "enum_field" {
				fieldName := child.ChildByFieldName("name")
				if fieldName == nil {
					for j := 0; j < int(child.ChildCount()); j++ {
						fieldChild := child.Child(j)
						if fieldChild != nil && fieldChild.Type() == "identifier" {
							fieldName = fieldChild
							break
						}
					}
				}

				if fieldName != nil {
					children = append(children, Symbol{
						Name:      fieldName.Content(content),
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
		Exported:   true,
		Children:   children,
	}
}

func (p *ProtobufParser) parseService(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "service_name" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractComment(node, content)
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "rpc" {
			rpcName := child.ChildByFieldName("name")
			if rpcName == nil {
				for j := 0; j < int(child.ChildCount()); j++ {
					rpcChild := child.Child(j)
					if rpcChild != nil && rpcChild.Type() == "rpc_name" {
						rpcName = rpcChild
						break
					}
				}
			}

			if rpcName != nil {
				children = append(children, Symbol{
					Name:      rpcName.Content(content),
					Kind:      KindMethod,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}

	return &Symbol{
		Name:       name,
		Kind:       KindInterface,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
		Children:   children,
	}
}

func (p *ProtobufParser) parsePackage(node *sitter.Node, content []byte) *Symbol {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "full_ident" || child.Type() == "identifier") {
			return &Symbol{
				Name:      child.Content(content),
				Kind:      KindType,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Signature: "package",
				Exported:  true,
			}
		}
	}

	return nil
}

func (p *ProtobufParser) parseOption(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var optionName string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "option_name" || child.Type() == "identifier" || child.Type() == "full_ident") {
			optionName = child.Content(content)
			break
		}
	}

	if optionName != "" {
		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      "option:" + optionName,
			Kind:      KindConstant,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
			Exported:  true,
		})
	}
}

func (p *ProtobufParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "import" {
			p.parseImport(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *ProtobufParser) parseImport(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "string" {
			path := strings.Trim(child.Content(content), "\"")
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: path,
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
			break
		}
	}
}

func (p *ProtobufParser) extractComment(node *sitter.Node, content []byte) string {
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
