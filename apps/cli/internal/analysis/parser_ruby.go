package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/ruby"
)

type RubyParser struct {
	parser *sitter.Parser
}

func NewRubyParser() *RubyParser {
	p := sitter.NewParser()
	p.SetLanguage(ruby.GetLanguage())
	return &RubyParser{parser: p}
}

func (p *RubyParser) Language() Language {
	return LangRuby
}

func (p *RubyParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangRuby),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *RubyParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "class":
			sym := p.parseClass(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "module":
			sym := p.parseModule(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "method":
			sym := p.parseMethod(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "singleton_method":
			sym := p.parseSingletonMethod(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "assignment":
			p.parseAssignment(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *RubyParser) parseClass(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractPrecedingComment(node, content)
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
		Exported:   true,
		Children:   children,
	}
}

func (p *RubyParser) parseModule(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractPrecedingComment(node, content)
	var children []Symbol

	body := node.ChildByFieldName("body")
	if body != nil {
		children = p.extractClassMembers(body, content)
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

func (p *RubyParser) parseMethod(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	doc := p.extractPrecedingComment(node, content)
	sig := p.extractMethodSignature(node, content)
	exported := !strings.HasPrefix(name, "_")

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

func (p *RubyParser) parseSingletonMethod(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
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

func (p *RubyParser) parseAssignment(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	left := node.ChildByFieldName("left")
	if left == nil {
		return
	}

	if left.Type() == "constant" {
		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      left.Content(content),
			Kind:      KindConstant,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
			Exported:  true,
		})
	}
}

func (p *RubyParser) extractClassMembers(node *sitter.Node, content []byte) []Symbol {
	var members []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "method":
			sym := p.parseMethod(child, content)
			if sym != nil {
				members = append(members, *sym)
			}

		case "singleton_method":
			sym := p.parseSingletonMethod(child, content)
			if sym != nil {
				members = append(members, *sym)
			}

		case "call":
			p.parseAttrAccessor(child, content, &members)
		}
	}
	return members
}

func (p *RubyParser) parseAttrAccessor(node *sitter.Node, content []byte, members *[]Symbol) {
	methodNode := node.ChildByFieldName("method")
	if methodNode == nil {
		return
	}

	method := methodNode.Content(content)
	if method != "attr_reader" && method != "attr_writer" && method != "attr_accessor" {
		return
	}

	args := node.ChildByFieldName("arguments")
	if args == nil {
		return
	}

	for i := 0; i < int(args.ChildCount()); i++ {
		arg := args.Child(i)
		if arg != nil && (arg.Type() == "simple_symbol" || arg.Type() == "symbol") {
			name := strings.TrimPrefix(arg.Content(content), ":")
			*members = append(*members, Symbol{
				Name:      name,
				Kind:      KindProperty,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Exported:  true,
			})
		}
	}
}

func (p *RubyParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "call":
			p.parseRequire(child, content, analysis)

		case "class":
			p.parseInheritance(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *RubyParser) parseRequire(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	methodNode := node.ChildByFieldName("method")
	if methodNode == nil {
		return
	}

	method := methodNode.Content(content)
	if method != "require" && method != "require_relative" {
		return
	}

	args := node.ChildByFieldName("arguments")
	if args == nil {
		return
	}

	for i := 0; i < int(args.ChildCount()); i++ {
		arg := args.Child(i)
		if arg != nil && arg.Type() == "string" {
			path := strings.Trim(arg.Content(content), "\"'")
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: path,
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
		}
	}
}

func (p *RubyParser) parseInheritance(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	superclass := node.ChildByFieldName("superclass")
	if superclass != nil {
		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetSymbol: superclass.Content(content),
			Kind:         RelExtends,
			Line:         int(node.StartPoint().Row) + 1,
		})
	}
}

func (p *RubyParser) extractMethodSignature(node *sitter.Node, content []byte) string {
	nameNode := node.ChildByFieldName("name")
	paramsNode := node.ChildByFieldName("parameters")

	var sig strings.Builder
	sig.WriteString("def ")
	if nameNode != nil {
		sig.WriteString(nameNode.Content(content))
	}
	if paramsNode != nil {
		sig.WriteString(paramsNode.Content(content))
	}

	return sig.String()
}

func (p *RubyParser) extractPrecedingComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		return strings.TrimPrefix(prev.Content(content), "# ")
	}

	return ""
}
