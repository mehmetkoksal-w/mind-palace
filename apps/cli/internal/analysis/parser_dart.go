package analysis

import (
	"regexp"
	"strings"
)

// DartParser uses regex-based parsing since there are no stable
// Tree-sitter Go bindings for Dart.
type DartParser struct{}

func NewDartParser() *DartParser {
	return &DartParser{}
}

func (p *DartParser) Language() Language {
	return LangDart
}

var (
	dartClassRegex      = regexp.MustCompile(`(?m)^(\s*)(abstract\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?(?:\s+(?:with|implements)\s+[\w\s,]+)?\s*\{`)
	dartMixinRegex      = regexp.MustCompile(`(?m)^(\s*)mixin\s+(\w+)(?:\s+on\s+[\w\s,]+)?\s*\{`)
	dartEnumRegex       = regexp.MustCompile(`(?m)^(\s*)enum\s+(\w+)\s*\{`)
	dartFunctionRegex   = regexp.MustCompile(`(?m)^(\s*)([\w<>\[\]?,\s]+)\s+(\w+)\s*\(([^)]*)\)\s*(async\s*)?\{`)
	dartMethodRegex     = regexp.MustCompile(`(?m)^(\s+)(static\s+)?([\w<>\[\]?,\s]+)\s+(\w+)\s*\(([^)]*)\)\s*(async\s*)?\{`)
	dartGetterRegex     = regexp.MustCompile(`(?m)^(\s+)(static\s+)?([\w<>\[\]?,\s]+)\s+get\s+(\w+)\s*[{=>]`)
	dartSetterRegex     = regexp.MustCompile(`(?m)^(\s+)set\s+(\w+)\s*\(`)
	dartFieldRegex      = regexp.MustCompile(`(?m)^(\s+)(static\s+)?(final\s+)?(late\s+)?(const\s+)?([\w<>\[\]?,]+)\s+(\w+)\s*[;=]`)
	dartConstRegex      = regexp.MustCompile(`(?m)^(\s*)(const|final)\s+([\w<>\[\]?,]+)\s+(\w+)\s*=`)
	dartImportRegex     = regexp.MustCompile(`(?m)^import\s+['"]([^'"]+)['"]`)
	dartExportRegex     = regexp.MustCompile(`(?m)^export\s+['"]([^'"]+)['"]`)
	dartPartRegex       = regexp.MustCompile(`(?m)^part\s+['"]([^'"]+)['"]`)
	dartPartOfRegex     = regexp.MustCompile(`(?m)^part\s+of\s+['"]([^'"]+)['"]`)
	dartExtensionRegex  = regexp.MustCompile(`(?m)^(\s*)extension\s+(\w+)?\s+on\s+(\w+)`)
	dartTypedefRegex    = regexp.MustCompile(`(?m)^(\s*)typedef\s+(\w+)`)
	dartDocCommentRegex = regexp.MustCompile(`(?m)^(\s*)///\s*(.*)$`)
)

func (p *DartParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangDart),
	}

	lines := strings.Split(string(content), "\n")

	p.extractSymbols(lines, analysis)
	p.extractRelationships(lines, analysis)

	return analysis, nil
}

func (p *DartParser) extractSymbols(lines []string, analysis *FileAnalysis) {
	fullContent := strings.Join(lines, "\n")

	// Track class/mixin boundaries for method detection
	type blockInfo struct {
		name      string
		startLine int
		endLine   int
	}
	var blocks []blockInfo

	// Extract classes
	for _, match := range dartClassRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])
		nameStart, nameEnd := match[6], match[7]
		name := fullContent[nameStart:nameEnd]

		isAbstract := match[2] != -1 && match[3] != -1
		doc := p.extractDocComment(lines, lineNum-1)

		sym := Symbol{
			Name:       name,
			Kind:       KindClass,
			LineStart:  lineNum,
			LineEnd:    p.findBlockEnd(lines, lineNum-1),
			DocComment: doc,
			Exported:   !strings.HasPrefix(name, "_"),
		}

		if isAbstract {
			sym.Kind = KindInterface
		}

		blocks = append(blocks, blockInfo{name: name, startLine: sym.LineStart, endLine: sym.LineEnd})

		// Extract class body
		sym.Children = p.extractClassMembers(lines, sym.LineStart, sym.LineEnd)
		analysis.Symbols = append(analysis.Symbols, sym)
	}

	// Extract mixins
	for _, match := range dartMixinRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])
		nameStart, nameEnd := match[4], match[5]
		name := fullContent[nameStart:nameEnd]

		doc := p.extractDocComment(lines, lineNum-1)
		endLine := p.findBlockEnd(lines, lineNum-1)

		sym := Symbol{
			Name:       name,
			Kind:       KindClass,
			LineStart:  lineNum,
			LineEnd:    endLine,
			DocComment: doc,
			Exported:   !strings.HasPrefix(name, "_"),
		}

		blocks = append(blocks, blockInfo{name: name, startLine: sym.LineStart, endLine: sym.LineEnd})
		sym.Children = p.extractClassMembers(lines, sym.LineStart, sym.LineEnd)
		analysis.Symbols = append(analysis.Symbols, sym)
	}

	// Extract enums
	for _, match := range dartEnumRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])
		nameStart, nameEnd := match[4], match[5]
		name := fullContent[nameStart:nameEnd]

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:       name,
			Kind:       KindEnum,
			LineStart:  lineNum,
			LineEnd:    p.findBlockEnd(lines, lineNum-1),
			DocComment: p.extractDocComment(lines, lineNum-1),
			Exported:   !strings.HasPrefix(name, "_"),
		})
	}

	// Extract extensions
	for _, match := range dartExtensionRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])

		name := "extension"
		if match[4] != -1 && match[5] != -1 {
			name = fullContent[match[4]:match[5]]
		}

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      name,
			Kind:      KindClass,
			LineStart: lineNum,
			LineEnd:   p.findBlockEnd(lines, lineNum-1),
			Exported:  !strings.HasPrefix(name, "_"),
		})
	}

	// Extract typedefs
	for _, match := range dartTypedefRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])
		nameStart, nameEnd := match[4], match[5]
		name := fullContent[nameStart:nameEnd]

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      name,
			Kind:      KindType,
			LineStart: lineNum,
			LineEnd:   lineNum,
			Exported:  !strings.HasPrefix(name, "_"),
		})
	}

	// Extract top-level functions (not inside classes)
	for _, match := range dartFunctionRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])

		// Skip if inside a class block
		insideBlock := false
		for _, block := range blocks {
			if lineNum > block.startLine && lineNum < block.endLine {
				insideBlock = true
				break
			}
		}
		if insideBlock {
			continue
		}

		indent := fullContent[match[2]:match[3]]
		if len(strings.TrimSpace(indent)) > 0 {
			continue // Skip indented functions (methods)
		}

		returnType := strings.TrimSpace(fullContent[match[4]:match[5]])
		nameStart, nameEnd := match[6], match[7]
		name := fullContent[nameStart:nameEnd]
		params := fullContent[match[8]:match[9]]

		// Skip common keywords that look like return types but aren't
		if returnType == "if" || returnType == "for" || returnType == "while" || returnType == "switch" || returnType == "return" {
			continue
		}

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:       name,
			Kind:       KindFunction,
			LineStart:  lineNum,
			LineEnd:    p.findBlockEnd(lines, lineNum-1),
			Signature:  returnType + " " + name + "(" + params + ")",
			DocComment: p.extractDocComment(lines, lineNum-1),
			Exported:   !strings.HasPrefix(name, "_"),
		})
	}

	// Extract top-level constants
	for _, match := range dartConstRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])

		// Skip if inside a class block
		insideBlock := false
		for _, block := range blocks {
			if lineNum > block.startLine && lineNum < block.endLine {
				insideBlock = true
				break
			}
		}
		if insideBlock {
			continue
		}

		indent := fullContent[match[2]:match[3]]
		if len(strings.TrimSpace(indent)) > 0 {
			continue
		}

		nameStart, nameEnd := match[8], match[9]
		name := fullContent[nameStart:nameEnd]

		analysis.Symbols = append(analysis.Symbols, Symbol{
			Name:      name,
			Kind:      KindConstant,
			LineStart: lineNum,
			LineEnd:   lineNum,
			Exported:  !strings.HasPrefix(name, "_"),
		})
	}
}

func (p *DartParser) extractClassMembers(lines []string, startLine, endLine int) []Symbol {
	var children []Symbol

	if startLine < 1 || endLine > len(lines) {
		return children
	}

	classContent := strings.Join(lines[startLine-1:endLine], "\n")

	// Extract methods
	for _, match := range dartMethodRegex.FindAllStringSubmatchIndex(classContent, -1) {
		localLineNum := p.lineNumberAt(classContent, match[0])
		lineNum := startLine + localLineNum - 1

		isStatic := match[4] != -1 && match[5] != -1
		returnType := strings.TrimSpace(classContent[match[6]:match[7]])
		name := classContent[match[8]:match[9]]
		params := classContent[match[10]:match[11]]

		// Skip common keywords
		if returnType == "if" || returnType == "for" || returnType == "while" || returnType == "switch" || returnType == "return" {
			continue
		}

		kind := KindMethod
		if name == "constructor" || strings.Contains(name, ".") {
			kind = KindConstructor
		}

		sig := returnType + " " + name + "(" + params + ")"
		if isStatic {
			sig = "static " + sig
		}

		children = append(children, Symbol{
			Name:      name,
			Kind:      kind,
			LineStart: lineNum,
			LineEnd:   lineNum,
			Signature: sig,
			Exported:  !strings.HasPrefix(name, "_"),
		})
	}

	// Extract getters
	for _, match := range dartGetterRegex.FindAllStringSubmatchIndex(classContent, -1) {
		localLineNum := p.lineNumberAt(classContent, match[0])
		lineNum := startLine + localLineNum - 1

		name := classContent[match[8]:match[9]]
		returnType := strings.TrimSpace(classContent[match[6]:match[7]])

		children = append(children, Symbol{
			Name:      name,
			Kind:      KindProperty,
			LineStart: lineNum,
			LineEnd:   lineNum,
			Signature: returnType + " get " + name,
			Exported:  !strings.HasPrefix(name, "_"),
		})
	}

	// Extract setters
	for _, match := range dartSetterRegex.FindAllStringSubmatchIndex(classContent, -1) {
		localLineNum := p.lineNumberAt(classContent, match[0])
		lineNum := startLine + localLineNum - 1

		name := classContent[match[4]:match[5]]

		children = append(children, Symbol{
			Name:      name,
			Kind:      KindProperty,
			LineStart: lineNum,
			LineEnd:   lineNum,
			Signature: "set " + name,
			Exported:  !strings.HasPrefix(name, "_"),
		})
	}

	// Extract fields
	for _, match := range dartFieldRegex.FindAllStringSubmatchIndex(classContent, -1) {
		localLineNum := p.lineNumberAt(classContent, match[0])
		lineNum := startLine + localLineNum - 1

		isStatic := match[4] != -1 && match[5] != -1
		isFinal := match[6] != -1 && match[7] != -1
		isConst := match[10] != -1 && match[11] != -1

		fieldType := strings.TrimSpace(classContent[match[12]:match[13]])
		name := classContent[match[14]:match[15]]

		kind := KindProperty
		if isConst || (isStatic && isFinal) {
			kind = KindConstant
		}

		sig := fieldType + " " + name
		if isStatic {
			sig = "static " + sig
		}

		children = append(children, Symbol{
			Name:      name,
			Kind:      kind,
			LineStart: lineNum,
			LineEnd:   lineNum,
			Signature: sig,
			Exported:  !strings.HasPrefix(name, "_"),
		})
	}

	return children
}

func (p *DartParser) extractRelationships(lines []string, analysis *FileAnalysis) {
	fullContent := strings.Join(lines, "\n")

	// Extract imports
	for _, match := range dartImportRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])
		importPath := fullContent[match[2]:match[3]]

		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetFile: importPath,
			Kind:       RelImport,
			Line:       lineNum,
		})
	}

	// Extract exports
	for _, match := range dartExportRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])
		exportPath := fullContent[match[2]:match[3]]

		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetFile: exportPath,
			Kind:       RelImport,
			Line:       lineNum,
		})
	}

	// Extract part statements
	for _, match := range dartPartRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])
		partPath := fullContent[match[2]:match[3]]

		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetFile: partPath,
			Kind:       RelImport,
			Line:       lineNum,
		})
	}

	// Extract part of statements
	for _, match := range dartPartOfRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])
		partOfPath := fullContent[match[2]:match[3]]

		analysis.Relationships = append(analysis.Relationships, Relationship{
			TargetFile: partOfPath,
			Kind:       RelImport,
			Line:       lineNum,
		})
	}

	// Extract extends relationships from class declarations
	for _, match := range dartClassRegex.FindAllStringSubmatchIndex(fullContent, -1) {
		lineNum := p.lineNumberAt(fullContent, match[0])

		// Check if extends is present
		if match[8] != -1 && match[9] != -1 {
			parentClass := fullContent[match[8]:match[9]]
			analysis.Relationships = append(analysis.Relationships, Relationship{
				TargetSymbol: parentClass,
				Kind:         RelExtends,
				Line:         lineNum,
			})
		}
	}
}

func (p *DartParser) lineNumberAt(content string, pos int) int {
	return strings.Count(content[:pos], "\n") + 1
}

func (p *DartParser) findBlockEnd(lines []string, startIdx int) int {
	if startIdx < 0 || startIdx >= len(lines) {
		return startIdx + 1
	}

	braceCount := 0
	started := false

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		for _, ch := range line {
			if ch == '{' {
				braceCount++
				started = true
			} else if ch == '}' {
				braceCount--
				if started && braceCount == 0 {
					return i + 1
				}
			}
		}
	}

	return startIdx + 1
}

func (p *DartParser) extractDocComment(lines []string, lineIdx int) string {
	if lineIdx < 0 || lineIdx >= len(lines) {
		return ""
	}

	var docLines []string

	for i := lineIdx - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "///") {
			doc := strings.TrimPrefix(line, "///")
			doc = strings.TrimSpace(doc)
			docLines = append([]string{doc}, docLines...)
		} else if line == "" {
			continue
		} else {
			break
		}
	}

	return strings.Join(docLines, " ")
}
