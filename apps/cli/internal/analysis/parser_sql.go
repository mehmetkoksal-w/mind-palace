package analysis

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/sql"
)

type SQLParser struct {
	parser *sitter.Parser
}

func NewSQLParser() *SQLParser {
	p := sitter.NewParser()
	p.SetLanguage(sql.GetLanguage())
	return &SQLParser{parser: p}
}

func (p *SQLParser) Language() Language {
	return LangSQL
}

func (p *SQLParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangSQL),
	}

	root := tree.RootNode()
	p.extractSymbols(root, content, analysis)
	p.extractRelationships(root, content, analysis)

	return analysis, nil
}

func (p *SQLParser) extractSymbols(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "create_table":
			sym := p.parseCreateTable(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "create_function":
			sym := p.parseCreateFunction(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "create_view":
			sym := p.parseCreateView(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "create_index":
			sym := p.parseCreateIndex(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "create_trigger":
			sym := p.parseCreateTrigger(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "create_procedure":
			sym := p.parseCreateProcedure(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}

		case "create_type":
			sym := p.parseCreateType(child, content)
			if sym != nil {
				analysis.Symbols = append(analysis.Symbols, *sym)
			}
		}

		p.extractSymbols(child, content, analysis)
	}
}

func (p *SQLParser) parseCreateTable(node *sitter.Node, content []byte) *Symbol {
	name := p.extractObjectName(node, content)
	if name == "" {
		return nil
	}

	doc := p.extractPrecedingComment(node, content)
	var children []Symbol

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "column_definitions" {
			children = p.extractColumns(child, content)
			break
		}
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

func (p *SQLParser) parseCreateFunction(node *sitter.Node, content []byte) *Symbol {
	name := p.extractObjectName(node, content)
	if name == "" {
		return nil
	}

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

func (p *SQLParser) parseCreateView(node *sitter.Node, content []byte) *Symbol {
	name := p.extractObjectName(node, content)
	if name == "" {
		return nil
	}

	doc := p.extractPrecedingComment(node, content)

	return &Symbol{
		Name:       name,
		Kind:       KindType,
		LineStart:  int(node.StartPoint().Row) + 1,
		LineEnd:    int(node.EndPoint().Row) + 1,
		DocComment: doc,
		Exported:   true,
	}
}

func (p *SQLParser) parseCreateIndex(node *sitter.Node, content []byte) *Symbol {
	name := p.extractObjectName(node, content)
	if name == "" {
		return nil
	}

	return &Symbol{
		Name:      name,
		Kind:      KindVariable,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Exported:  true,
	}
}

func (p *SQLParser) parseCreateTrigger(node *sitter.Node, content []byte) *Symbol {
	name := p.extractObjectName(node, content)
	if name == "" {
		return nil
	}

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

func (p *SQLParser) parseCreateProcedure(node *sitter.Node, content []byte) *Symbol {
	name := p.extractObjectName(node, content)
	if name == "" {
		return nil
	}

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

func (p *SQLParser) parseCreateType(node *sitter.Node, content []byte) *Symbol {
	name := p.extractObjectName(node, content)
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

func (p *SQLParser) extractObjectName(node *sitter.Node, content []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "identifier", "object_reference", "table_reference":
			name := child.Content(content)
			name = strings.Trim(name, "`\"[]")
			if strings.Contains(name, ".") {
				parts := strings.Split(name, ".")
				name = parts[len(parts)-1]
			}
			return name
		}
	}
	return ""
}

func (p *SQLParser) extractColumns(node *sitter.Node, content []byte) []Symbol {
	var columns []Symbol
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "column_definition" {
			var name string
			for j := 0; j < int(child.ChildCount()); j++ {
				col := child.Child(j)
				if col != nil && col.Type() == "identifier" {
					name = strings.Trim(col.Content(content), "`\"[]")
					break
				}
			}

			if name != "" {
				columns = append(columns, Symbol{
					Name:      name,
					Kind:      KindProperty,
					LineStart: int(child.StartPoint().Row) + 1,
					LineEnd:   int(child.EndPoint().Row) + 1,
					Exported:  true,
				})
			}
		}
	}
	return columns
}

func (p *SQLParser) extractRelationships(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "from_clause", "join_clause":
			p.parseTableReference(child, content, analysis)

		case "foreign_key_constraint":
			p.parseForeignKey(child, content, analysis)
		}

		p.extractRelationships(child, content, analysis)
	}
}

func (p *SQLParser) parseTableReference(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "identifier" || child.Type() == "object_reference" || child.Type() == "table_reference" {
			tableName := strings.Trim(child.Content(content), "`\"[]")
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetSymbol: tableName,
				Kind:         RelReference,
				Line:         int(child.StartPoint().Row) + 1,
			})
		}
	}
}

func (p *SQLParser) parseForeignKey(node *sitter.Node, content []byte, analysis *FileAnalysis) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "references_constraint" {
			for j := 0; j < int(child.ChildCount()); j++ {
				ref := child.Child(j)
				if ref != nil && (ref.Type() == "identifier" || ref.Type() == "object_reference") {
					tableName := strings.Trim(ref.Content(content), "`\"[]")
					analysis.Relationships = append(analysis.Relationships, Relationship{
						TargetSymbol: tableName,
						Kind:         RelReference,
						Line:         int(ref.StartPoint().Row) + 1,
					})
					break
				}
			}
		}
	}
}

func (p *SQLParser) extractPrecedingComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		comment := prev.Content(content)
		comment = strings.TrimPrefix(comment, "-- ")
		comment = strings.TrimPrefix(comment, "/* ")
		comment = strings.TrimSuffix(comment, " */")
		return comment
	}

	return ""
}
