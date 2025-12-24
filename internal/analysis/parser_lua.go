package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/lua"
)

type LuaParser struct {
	parser *sitter.Parser
}

func NewLuaParser() *LuaParser {
	p := sitter.NewParser()
	p.SetLanguage(lua.GetLanguage())
	return &LuaParser{parser: p}
}

func (p *LuaParser) Language() Language {
	return LangLua
}

func (p *LuaParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangLua),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *LuaParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_declaration":
			sym := p.parseFunctionDecl(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "local_function":
			sym := p.parseLocalFunction(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "variable_declaration":
			p.parseVariableDecl(child, content, analysis)

		case "local_variable_declaration":
			p.parseLocalVariableDecl(child, content, analysis)

		case "assignment_statement":
			p.parseAssignment(child, content, analysis)
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *LuaParser) parseFunctionDecl(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && (child.Type() == "identifier" || child.Type() == "dot_index_expression" || child.Type() == "method_index_expression") {
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
	sig := p.extractFunctionSignature(node, content)

	kind := KindFunction
	if strings.Contains(name, ":") || strings.Contains(name, ".") {
		kind = KindMethod
	}

	return &Symbol{
		Name:       name,
		Kind:       kind,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  sig,
		DocComment: doc,
		Exported:   true,
	}
}

func (p *LuaParser) parseLocalFunction(node *sitter.Node, content []byte) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "identifier" {
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
	sig := p.extractFunctionSignature(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindFunction,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		Signature:  sig,
		DocComment: doc,
		Exported:   false,
	}
}

func (p *LuaParser) parseVariableDecl(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "variable_list" {
			for j := 0; j < int(child.ChildCount()); j++ {
				varNode := child.Child(j)
				if varNode != nil && varNode.Type() == "identifier" {
					analysis.Symbols = append(analysis.Symbols, Symbol{
						Name:      varNode.Content(content),
						Kind:      KindVariable,
						LineStart: int(node.StartPoint().Row) + 1,
						LineEnd:   int(node.EndPoint().Row) + 1,
						Exported:  true,
					})
				}
			}
		}
	}
}

func (p *LuaParser) parseLocalVariableDecl(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "variable_list" {
			for j := 0; j < int(child.ChildCount()); j++ {
				varNode := child.Child(j)
				if varNode != nil && varNode.Type() == "identifier" {
					analysis.Symbols = append(analysis.Symbols, Symbol{
						Name:      varNode.Content(content),
						Kind:      KindVariable,
						LineStart: int(node.StartPoint().Row) + 1,
						LineEnd:   int(node.EndPoint().Row) + 1,
						Exported:  false,
					})
				}
			}
		}
	}
}

func (p *LuaParser) parseAssignment(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "variable_list" {
			for j := 0; j < int(child.ChildCount()); j++ {
				varNode := child.Child(j)
				if varNode == nil {
					continue
				}

				if varNode.Type() == "identifier" {
					name := varNode.Content(content)
					if strings.ToUpper(name) == name {
						analysis.Symbols = append(analysis.Symbols, Symbol{
							Name:      name,
							Kind:      KindConstant,
							LineStart: int(node.StartPoint().Row) + 1,
							LineEnd:   int(node.EndPoint().Row) + 1,
							Exported:  true,
						})
					}
				}

				if varNode.Type() == "dot_index_expression" {
					varContent := varNode.Content(content)
					if strings.Contains(varContent, ".") {
						parts := strings.Split(varContent, ".")
						if len(parts) == 2 {
							analysis.Symbols = append(analysis.Symbols, Symbol{
								Name:      varContent,
								Kind:      KindProperty,
								LineStart: int(node.StartPoint().Row) + 1,
								LineEnd:   int(node.EndPoint().Row) + 1,
								Exported:  true,
							})
						}
					}
				}
			}
		}
	}
}

func (p *LuaParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "function_call" {
			p.parseRequireCall(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *LuaParser) parseRequireCall(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	var funcName string
	var args *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "identifier":
			funcName = child.Content(content)
		case "arguments":
			args = child
		}
	}

	if funcName != "require" && funcName != "dofile" && funcName != "loadfile" {
		return
	}

	if args == nil {
		return
	}

	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		if child != nil && child.Type() == "string" {
			modulePath := strings.Trim(child.Content(content), "\"'")
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetFile: modulePath,
				Kind:       RelImport,
				Line:       int(node.StartPoint().Row) + 1,
			})
			break
		}
	}
}

func (p *LuaParser) extractFunctionSignature(node *sitter.Node, content []byte) string {
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "parameters" {
				paramsNode = child
				break
			}
		}
	}

	if paramsNode != nil {
		return "function" + paramsNode.Content(content)
	}

	return "function()"
}

func (p *LuaParser) extractPrecedingComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		comment := prev.Content(content)
		comment = strings.TrimPrefix(comment, "-- ")
		comment = strings.TrimPrefix(comment, "--[[ ")
		comment = strings.TrimSuffix(comment, " ]]")
		return comment
	}

	return ""
}
