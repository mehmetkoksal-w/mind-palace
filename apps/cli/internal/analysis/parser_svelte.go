package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/svelte"
)

type SvelteParser struct {
	parser *sitter.Parser
}

func NewSvelteParser() *SvelteParser {
	p := sitter.NewParser()
	p.SetLanguage(svelte.GetLanguage())
	return &SvelteParser{parser: p}
}

func (p *SvelteParser) Language() Language {
	return LangSvelte
}

func (p *SvelteParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangSvelte),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *SvelteParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "script_element":
			p.parseScriptElement(child, content, analysis)

		case "style_element":
			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:      "<style>",
				Kind:      KindType,
				LineStart: int(child.StartPoint().Row) + 1,
				LineEnd:   int(child.EndPoint().Row) + 1,
				Exported:  true,
			})

		case "element":
			p.parseElement(child, content, analysis)

		case "each_statement":
			p.parseEachStatement(child, content, analysis)

		case "if_statement":
			p.parseIfStatement(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *SvelteParser) parseScriptElement(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	isModule := false

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "start_tag" {
			for j := 0; j < int(child.ChildCount()); j++ {
				attr := child.Child(j)
				if attr != nil && attr.Type() == "attribute" {
					attrContent := attr.Content(content)
					if strings.Contains(attrContent, "context=\"module\"") || strings.Contains(attrContent, "context='module'") {
						isModule = true
					}
				}
			}
		}

		if child.Type() == "raw_text" {
			p.parseScriptContent(child, content, analysis, isModule)
		}
	}

	kind := KindClass
	name := "<script>"
	if isModule {
		name = "<script context=\"module\">"
		kind = KindInterface
	}

	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      name,
		Kind:      kind,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	})
}

func (p *SvelteParser) parseScriptContent(node *sitter.Node, content []byte, analysis *FileAnalysis, _ bool) {
	text := node.Content(content)

	lines := strings.Split(text, "\n")
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "export let ") || strings.HasPrefix(line, "export const ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				name := strings.TrimSuffix(parts[2], ";")
				name = strings.Split(name, "=")[0]
				name = strings.TrimSpace(name)

				kind := KindProperty
				if strings.HasPrefix(line, "export const ") {
					kind = KindConstant
				}

				analysis.Symbols = append(analysis.Symbols, Symbol{
					Name:      name,
					Kind:      kind,
					LineStart: int(node.StartPoint().Row) + lineNum + 1,
					LineEnd:   int(node.StartPoint().Row) + lineNum + 1,
					Exported:  true,
				})
			}
		}

		if strings.HasPrefix(line, "function ") {
			parts := strings.Split(line, "(")
			if len(parts) > 0 {
				name := strings.TrimPrefix(parts[0], "function ")
				name = strings.TrimSpace(name)

				analysis.Symbols = append(analysis.Symbols, Symbol{
					Name:      name,
					Kind:      KindFunction,
					LineStart: int(node.StartPoint().Row) + lineNum + 1,
					LineEnd:   int(node.StartPoint().Row) + lineNum + 1,
					Exported:  false,
				})
			}
		}
	}
}

func (p *SvelteParser) parseElement(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var tagName string
	var slotName string
	var bindName string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "start_tag" {
			for j := 0; j < int(child.ChildCount()); j++ {
				tagChild := child.Child(j)
				if tagChild == nil {
					continue
				}

				if tagChild.Type() == "tag_name" {
					tagName = tagChild.Content(content)
				}

				if tagChild.Type() == "attribute" {
					attrContent := tagChild.Content(content)
					if strings.HasPrefix(attrContent, "name=") {
						slotName = strings.Trim(strings.TrimPrefix(attrContent, "name="), "\"'")
					}
					if strings.HasPrefix(attrContent, "bind:") {
						bindName = strings.Split(attrContent, "=")[0]
						bindName = strings.TrimPrefix(bindName, "bind:")
					}
				}
			}
		}
	}

	if tagName == "slot" && slotName != "" {
		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      "slot:" + slotName,
			Kind:      KindInterface,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
			Exported:  true,
		})
	}

	if bindName != "" {
		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      "bind:" + bindName,
			Kind:      KindVariable,
			LineStart: int(node.StartPoint().Row) + 1,
			LineEnd:   int(node.EndPoint().Row) + 1,
			Exported:  false,
		})
	}
}

func (p *SvelteParser) parseEachStatement(node *sitter.Node, _ []byte, analysis *FileAnalysis) {
	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      "{#each}",
		Kind:      KindFunction,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  false,
	})
}

func (p *SvelteParser) parseIfStatement(node *sitter.Node, _ []byte, analysis *FileAnalysis) {
	analysis.Symbols = append(analysis.Symbols, Symbol{
		Name:      "{#if}",
		Kind:      KindFunction,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  false,
	})
}

func (p *SvelteParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "script_element":
			p.parseScriptImports(child, content, analysis)

		case "element":
			p.parseComponentUsage(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *SvelteParser) parseScriptImports(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "raw_text" {
			text := child.Content(content)
			lines := strings.Split(text, "\n")

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "import ") {
					parts := strings.Split(line, " from ")
					if len(parts) == 2 {
						path := strings.TrimSuffix(parts[1], ";")
						path = strings.Trim(path, "\"'")

						analysis.Relationships = append(analysis.Relationships, Relationship{
							TargetFile: path,
							Kind:       RelImport,
							Line:       int(child.StartPoint().Row) + 1,
						})
					}
				}
			}
		}
	}
}

func (p *SvelteParser) parseComponentUsage(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "start_tag" {
			for j := 0; j < int(child.ChildCount()); j++ {
				tagChild := child.Child(j)
				if tagChild != nil && tagChild.Type() == "tag_name" {
					tagName := tagChild.Content(content)
					if tagName != "" && tagName[0] >= 'A' && tagName[0] <= 'Z' {
						analysis.Relationships = append(analysis.Relationships, Relationship{
							TargetSymbol: tagName,
							Kind:         RelUses,
							Line:         int(node.StartPoint().Row) + 1,
						})
					}
				}
			}
		}
	}
}
