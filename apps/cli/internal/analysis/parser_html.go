package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/html"
)

type HTMLParser struct {
	parser *sitter.Parser
}

func NewHTMLParser() *HTMLParser {
	p := sitter.NewParser()
	p.SetLanguage(html.GetLanguage())
	return &HTMLParser{parser: p}
}

func (p *HTMLParser) Language() Language {
	return LangHTML
}

func (p *HTMLParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangHTML),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *HTMLParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "element" {
			p.parseElement(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *HTMLParser) parseElement(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var tagName string
	var id string
	var classes []string

	startTag := node.ChildByFieldName("start_tag")
	if startTag == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "start_tag" {
				startTag = child
				break
			}
		}
	}

	if startTag == nil {
		return
	}

	for i := 0; i < int(startTag.ChildCount()); i++ {
		child := startTag.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "tag_name":
			tagName = child.Content(content)
		case "attribute":
			attrName, attrValue := p.parseAttribute(child, content)
			switch attrName {
			case "id":
				id = attrValue
			case "class":
				classes = strings.Fields(attrValue)
			}
		}
	}

	if id != "" {
		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      "#" + id,
			Kind:      KindVariable,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
			Signature: tagName,
			Exported:  true,
		})
	}

	for _, class := range classes {
		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      "." + class,
			Kind:      KindType,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
			Signature: tagName,
			Exported:  true,
		})
	}

	if tagName == "template" || tagName == "slot" {
		name := id
		if name == "" && len(classes) > 0 {
			name = classes[0]
		}
		if name == "" {
			name = tagName
		}

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      name,
			Kind:      KindInterface,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
			Exported:  true,
		})
	}
}

func (p *HTMLParser) parseAttribute(node *sitter.Node, content []byte) (string, string) {
	var name, value string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "attribute_name":
			name = child.Content(content)
		case "attribute_value", "quoted_attribute_value":
			value = strings.Trim(child.Content(content), "\"'")
		}
	}
	return name, value
}

func (p *HTMLParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "element" {
			p.parseElementRelationships(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *HTMLParser) parseElementRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	startTag := node.ChildByFieldName("start_tag")
	if startTag == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "start_tag" {
				startTag = child
				break
			}
		}
	}

	if startTag == nil {
		return
	}

	var tagName string
	for i := 0; i < int(startTag.ChildCount()); i++ {
		child := startTag.Child(i)
		if child != nil && child.Type() == "tag_name" {
			tagName = child.Content(content)
			break
		}
	}

	for i := 0; i < int(startTag.ChildCount()); i++ {
		child := startTag.Child(i)
		if child == nil || child.Type() != "attribute" {
			continue
		}

		attrName, attrValue := p.parseAttribute(child, content)

		switch tagName {
		case "script":
			if attrName == "src" && attrValue != "" {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: attrValue,
					Kind:       RelImport,
					Line:       int(child.StartPoint().Row) + 1,
				})
			}
		case "link":
			if attrName == "href" && attrValue != "" {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: attrValue,
					Kind:       RelImport,
					Line:       int(child.StartPoint().Row) + 1,
				})
			}
		case "img", "video", "audio", "source", "iframe":
			if attrName == "src" && attrValue != "" {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: attrValue,
					Kind:       RelReference,
					Line:       int(child.StartPoint().Row) + 1,
				})
			}
		case "a":
			if attrName == "href" && attrValue != "" && !strings.HasPrefix(attrValue, "#") && !strings.HasPrefix(attrValue, "http") {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: attrValue,
					Kind:       RelReference,
					Line:       int(child.StartPoint().Row) + 1,
				})
			}
		}
	}
}
