package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/dockerfile"
)

type DockerfileParser struct {
	parser *sitter.Parser
}

func NewDockerfileParser() *DockerfileParser {
	p := sitter.NewParser()
	p.SetLanguage(dockerfile.GetLanguage())
	return &DockerfileParser{parser: p}
}

func (p *DockerfileParser) Language() Language {
	return LangDockerfile
}

func (p *DockerfileParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangDockerfile),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *DockerfileParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	stageCount := 0

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "from_instruction":
			sym := p.parseFromInstruction(child, content, &stageCount)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "arg_instruction":
			p.parseArgInstruction(child, content, analysis)

		case "env_instruction":
			p.parseEnvInstruction(child, content, analysis)

		case "label_instruction":
			p.parseLabelInstruction(child, content, analysis)

		case "expose_instruction":
			p.parseExposeInstruction(child, content, analysis)

		case "entrypoint_instruction", "cmd_instruction":
			sym := p.parseEntrypoint(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *DockerfileParser) parseFromInstruction(node *sitter.Node, content []byte, stageCount *int) *Symbol {
	var image string
	var alias string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "image_spec":
			image = child.Content(content)
		case "image_alias":
			alias = child.Content(content)
		}
	}

	if image == "" {
		return nil
	}

	name := alias
	if name == "" {
		*stageCount++
		name = image
	}

	return &Symbol{
		Name:      name,
		Kind:      KindClass,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: "FROM " + image,
		Exported:  true,
	}
}

func (p *DockerfileParser) parseArgInstruction(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "unquoted_string" || child.Type() == "arg_name" {
			name := child.Content(content)
			if idx := strings.Index(name, "="); idx > 0 {
				name = name[:idx]
			}

			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:      name,
				Kind:      KindVariable,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Exported:  true,
			})
			break
		}
	}
}

func (p *DockerfileParser) parseEnvInstruction(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "env_pair" {
			var name string
			for j := 0; j < int(child.ChildCount()); j++ {
				pair := child.Child(j)
				if pair != nil && pair.Type() == "unquoted_string" {
					name = pair.Content(content)
					break
				}
			}

			if name != "" {
				analysis.Symbols = append(analysis.Symbols, Symbol{
					Name:      name,
					Kind:      KindConstant,
					LineStart: int(node.StartPoint().Row) + 1,
					LineEnd:   int(node.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}
}

func (p *DockerfileParser) parseLabelInstruction(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "label_pair" {
			var name string
			for j := 0; j < int(child.ChildCount()); j++ {
				pair := child.Child(j)
				if pair != nil && (pair.Type() == "unquoted_string" || pair.Type() == "double_quoted_string") {
					name = strings.Trim(pair.Content(content), "\"")
					break
				}
			}

			if name != "" {
				analysis.Symbols = append(analysis.Symbols, Symbol{
					Name:      name,
					Kind:      KindProperty,
					LineStart: int(node.StartPoint().Row) + 1,
					LineEnd:   int(node.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}
}

func (p *DockerfileParser) parseExposeInstruction(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "expose_port" {
			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:      "PORT_" + child.Content(content),
				Kind:      KindConstant,
				LineStart: int(node.StartPoint().Row) + 1,
				LineEnd:   int(node.EndPoint().Row) + 1,
				Exported:  true,
			})
		}
	}
}

func (p *DockerfileParser) parseEntrypoint(node *sitter.Node, _ []byte) *Symbol {
	name := "entrypoint"
	if node.Type() == "cmd_instruction" {
		name = "cmd"
	}

	return &Symbol{
		Name:      name,
		Kind:      KindFunction,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	}
}

func (p *DockerfileParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "from_instruction":
			p.parseFromDependency(child, content, analysis)

		case "copy_instruction":
			p.parseCopyFrom(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *DockerfileParser) parseFromDependency(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "image_spec" {
			image := child.Content(content)
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: image,
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
			break
		}
	}
}

func (p *DockerfileParser) parseCopyFrom(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "param" {
			param := child.Content(content)
			if strings.HasPrefix(param, "--from=") {
				stageName := strings.TrimPrefix(param, "--from=")
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetSymbol: stageName,
					Kind:         RelReference,
					Line:         int(node.StartPoint().Row) + 1,
				})
			}
		}
	}
}
