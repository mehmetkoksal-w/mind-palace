package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	tree_sitter_markdown "github.com/smacker/go-tree-sitter/markdown/tree-sitter-markdown"
)

type MarkdownParser struct {
	parser *sitter.Parser
}

func NewMarkdownParser() *MarkdownParser {
	p := sitter.NewParser()
	p.SetLanguage(tree_sitter_markdown.GetLanguage())
	return &MarkdownParser{parser: p}
}

func (p *MarkdownParser) Language() Language {
	return LangMarkdown
}

func (p *MarkdownParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangMarkdown),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *MarkdownParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "atx_heading":
			sym := p.parseHeading(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "setext_heading":
			sym := p.parseSetextHeading(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "fenced_code_block":
			sym := p.parseCodeBlock(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "html_block":
			sym := p.parseHtmlBlock(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "link_reference_definition":
			sym := p.parseLinkDef(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *MarkdownParser) parseHeading(node *sitter.Node, content []byte) *Symbol {
	var level int
	var text string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "atx_h1_marker":
			level = 1
		case "atx_h2_marker":
			level = 2
		case "atx_h3_marker":
			level = 3
		case "atx_h4_marker":
			level = 4
		case "atx_h5_marker":
			level = 5
		case "atx_h6_marker":
			level = 6
		case "heading_content", "inline":
			text = strings.TrimSpace(child.Content(content))
		}
	}

	if text == "" {
		return nil
	}

	kind := KindClass
	if level > 2 {
		kind = KindMethod
	}

	return &Symbol{
		Name:      text,
		Kind:      kind,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: strings.Repeat("#", level),
		Exported:  true,
	}
}

func (p *MarkdownParser) parseSetextHeading(node *sitter.Node, content []byte) *Symbol {
	var text string
	level := 2

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "paragraph":
			text = strings.TrimSpace(child.Content(content))
		case "setext_h1_underline":
			level = 1
		case "setext_h2_underline":
			level = 2
		}
	}

	if text == "" {
		return nil
	}

	return &Symbol{
		Name:      text,
		Kind:      KindClass,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: strings.Repeat("#", level),
		Exported:  true,
	}
}

func (p *MarkdownParser) parseCodeBlock(node *sitter.Node, content []byte) *Symbol {
	var lang string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "info_string" {
			lang = strings.TrimSpace(child.Content(content))
			break
		}
	}

	if lang == "" {
		return nil
	}

	return &Symbol{
		Name:      "code:" + lang,
		Kind:      KindProperty,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: lang,
		Exported:  true,
	}
}

func (p *MarkdownParser) parseHtmlBlock(node *sitter.Node, content []byte) *Symbol {
	text := node.Content(content)

	if strings.Contains(text, "<script") || strings.Contains(text, "<style") {
		return &Symbol{
			Name:      "html-block",
			Kind:      KindType,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
			Exported:  true,
		}
	}

	return nil
}

func (p *MarkdownParser) parseLinkDef(node *sitter.Node, content []byte) *Symbol {
	var label string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "link_label" {
			label = strings.Trim(child.Content(content), "[]")
			break
		}
	}

	if label == "" {
		return nil
	}

	return &Symbol{
		Name:      "[" + label + "]",
		Kind:      KindVariable,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	}
}

func (p *MarkdownParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "inline_link", "full_reference_link", "collapsed_reference_link":
			p.parseLink(child, content, analysis)

		case "image":
			p.parseImage(child, content, analysis)

		case "link_reference_definition":
			p.parseLinkDefRelation(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *MarkdownParser) parseLink(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "link_destination" {
			url := child.Content(content)
			url = strings.Trim(url, "<>")

			if url != "" && !strings.HasPrefix(url, "#") && !strings.HasPrefix(url, "http") && !strings.HasPrefix(url, "mailto:") {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: url,
					Kind:       RelReference,
					Line:       int(node.StartPoint().Row) + 1,
				})
			}
			break
		}
	}
}

func (p *MarkdownParser) parseImage(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "link_destination" {
			url := child.Content(content)
			url = strings.Trim(url, "<>")

			if url != "" && !strings.HasPrefix(url, "http") && !strings.HasPrefix(url, "data:") {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: url,
					Kind:       RelReference,
					Line:       int(node.StartPoint().Row) + 1,
				})
			}
			break
		}
	}
}

func (p *MarkdownParser) parseLinkDefRelation(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "link_destination" {
			url := child.Content(content)
			url = strings.Trim(url, "<>")

			if url != "" && !strings.HasPrefix(url, "http") {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: url,
					Kind:       RelReference,
					Line:       int(node.StartPoint().Row) + 1,
				})
			}
			break
		}
	}
}
