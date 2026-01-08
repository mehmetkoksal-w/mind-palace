package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/elixir"
)

type ElixirParser struct {
	parser *sitter.Parser
}

func NewElixirParser() *ElixirParser {
	p := sitter.NewParser()
	p.SetLanguage(elixir.GetLanguage())
	return &ElixirParser{parser: p}
}

func (p *ElixirParser) Language() Language {
	return LangElixir
}

func (p *ElixirParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangElixir),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *ElixirParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "call" {
			p.parseCall(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *ElixirParser) parseCall(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var target string
	var args *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "identifier":
			if target == "" {
				target = child.Content(content)
			}
		case "arguments":
			args = child
		case "do_block":
			p.extractSymbols(child, content, analysis)
		}
	}

	switch target {
	case "defmodule":
		p.parseModule(node, args, content, analysis)
	case "def", "defp":
		p.parseFunction(node, args, content, analysis, target == "def")
	case "defmacro", "defmacrop":
		p.parseMacro(node, args, content, analysis, target == "defmacro")
	case "defstruct":
		p.parseStruct(node, args, content, analysis)
	case "defprotocol":
		p.parseProtocol(node, args, content, analysis)
	case "defimpl":
		p.parseImpl(node, args, content, analysis)
	case "@moduledoc", "@doc":
		// Skip doc attributes
	case "@":
		p.parseAttribute(node, content, analysis)
	}
}

func (p *ElixirParser) parseModule(node, args *sitter.Node, content []byte, analysis *FileAnalysis) {
	if args == nil {
		return
	}

	name := p.extractFirstArg(args, content)
	if name == "" {
		return
	}

	doc := p.extractModuledoc(node, content)
	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:       name,
		Kind:       KindClass,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
	})
}

func (p *ElixirParser) parseFunction(node, args *sitter.Node, content []byte, analysis *FileAnalysis, exported bool) {
	if args == nil {
		return
	}

	name := ""
	sig := ""

	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "call":
			for j := 0; j < int(child.ChildCount()); j++ {
				callChild := child.Child(j)
				if callChild != nil && callChild.Type() == "identifier" {
					name = callChild.Content(content)
					sig = child.Content(content)
					break
				}
			}
		case "identifier":
			if name == "" {
				name = child.Content(content)
			}
		}

		if name != "" {
			break
		}
	}

	if name == "" {
		return
	}

	doc := p.extractDoc(node, content)
	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:       name,
		Kind:       KindFunction,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  sig,
		DocComment: doc,
		Exported:   exported,
	})
}

func (p *ElixirParser) parseMacro(node, args *sitter.Node, content []byte, analysis *FileAnalysis, exported bool) {
	if args == nil {
		return
	}

	name := p.extractFirstArg(args, content)
	if name == "" {
		return
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      name,
		Kind:      KindFunction,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: "defmacro",
		Exported:  exported,
	})
}

func (p *ElixirParser) parseStruct(node, _ *sitter.Node, _ []byte, analysis *FileAnalysis) {
	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      "__struct__",
		Kind:      KindType,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	})
}

func (p *ElixirParser) parseProtocol(node, args *sitter.Node, content []byte, analysis *FileAnalysis) {
	if args == nil {
		return
	}

	name := p.extractFirstArg(args, content)
	if name == "" {
		return
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      name,
		Kind:      KindInterface,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	})
}

func (p *ElixirParser) parseImpl(node, args *sitter.Node, content []byte, analysis *FileAnalysis) {
	if args == nil {
		return
	}

	name := p.extractFirstArg(args, content)
	if name == "" {
		return
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      "impl:" + name,
		Kind:      KindClass,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	})
}

func (p *ElixirParser) parseAttribute(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	nodeContent := node.Content(content)
	if strings.HasPrefix(nodeContent, "@") {
		parts := strings.SplitN(nodeContent, " ", 2)
		if len(parts) > 0 {
			name := strings.TrimPrefix(parts[0], "@")
			if name != "moduledoc" && name != "doc" && name != "spec" && name != "type" {
				analysis.Symbols = append(analysis.Symbols, Symbol{
					Name:      "@" + name,
					Kind:      KindConstant,
					LineStart: int(node.StartPoint().Row) + 1,
					LineEnd:   int(node.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}
}

func (p *ElixirParser) extractFirstArg(args *sitter.Node, content []byte) string {
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "alias":
			return child.Content(content)
		case "atom":
			return strings.TrimPrefix(child.Content(content), ":")
		case "identifier":
			return child.Content(content)
		}
	}
	return ""
}

func (p *ElixirParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "call" {
			p.parseImportCall(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *ElixirParser) parseImportCall(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var target string
	var args *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "identifier":
			if target == "" {
				target = child.Content(content)
			}
		case "arguments":
			args = child
		}
	}

	if target != "import" && target != "alias" && target != "use" && target != "require" {
		return
	}

	if args == nil {
		return
	}

	moduleName := p.extractFirstArg(args, content)
	if moduleName != "" {
		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetFile: moduleName,
			Kind:       RelImport,
			Line:       int(node.StartPoint().Row) + 1,
		})
	}
}

func (p *ElixirParser) extractModuledoc(_ *sitter.Node, _ []byte) string {
	return ""
}

func (p *ElixirParser) extractDoc(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "call" {
		text := prev.Content(content)
		if strings.HasPrefix(text, "@doc") {
			return strings.TrimPrefix(text, "@doc ")
		}
	}

	return ""
}
