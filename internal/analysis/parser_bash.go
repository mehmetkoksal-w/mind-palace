package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
)

type BashParser struct {
	parser *sitter.Parser
}

func NewBashParser() *BashParser {
	p := sitter.NewParser()
	p.SetLanguage(bash.GetLanguage())
	return &BashParser{parser: p}
}

func (p *BashParser) Language() Language {
	return LangBash
}

func (p *BashParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangBash),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *BashParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_definition":
			sym := p.parseFunction(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "variable_assignment":
			p.parseVariable(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *BashParser) parseFunction(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "word" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractPrecedingComment(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindFunction,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
	}
}

func (p *BashParser) parseVariable(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "variable_name" {
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
		Exported:  true,
	})
}

func (p *BashParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "command" {
			p.parseSourceCommand(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *BashParser) parseSourceCommand(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "command_name" {
				nameNode = child
				break
			}
		}
	}

	if nameNode == nil {
		return
	}

	cmdName := nameNode.Content(content)
	if cmdName != "source" && cmdName != "." {
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "word" || child.Type() == "string") {
			path := strings.Trim(child.Content(content), "\"'")
			if path != cmdName {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: path,
					Kind:       RelImport,
					Line:       int(node.StartPoint().Row) + 1,
				})
				break
			}
		}
	}
}

func (p *BashParser) extractPrecedingComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		return strings.TrimPrefix(prev.Content(content), "# ")
	}

	return ""
}
