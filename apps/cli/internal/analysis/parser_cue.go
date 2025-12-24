package analysis

import (
	"regexp"
	"strings"
)

type CUEParser struct{}

func NewCUEParser() *CUEParser {
	return &CUEParser{}
}

func (p *CUEParser) Language() Language {
	return LangCUE
}

func (p *CUEParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangCUE),
	}

	lines := strings.Split(string(content), "\n")
	p.extractSymbols(lines, analysis)
	p.extractRelationships(lines, analysis)

	return analysis, nil
}

var (
	cuePackageRe    = regexp.MustCompile(`^package\s+(\w+)`)
	cueImportRe     = regexp.MustCompile(`^\s*"([^"]+)"`)
	cueDefinitionRe = regexp.MustCompile(`^#?(\w+):\s*\{`)
	cueFieldRe      = regexp.MustCompile(`^\s+(\w+):\s+`)
	cueLetRe        = regexp.MustCompile(`^let\s+(\w+)\s*=`)
	cueCommentRe    = regexp.MustCompile(`^//\s*(.*)`)
)

func (p *CUEParser) extractSymbols(lines []string, analysis *FileAnalysis) {
	var pendingDoc string
	inImport := false
	currentDef := ""
	defStartLine := 0
	braceCount := 0

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "import") {
			inImport = true
			continue
		}

		if inImport {
			if strings.Contains(trimmed, ")") || !strings.Contains(trimmed, "\"") && !strings.HasPrefix(trimmed, "\"") {
				inImport = false
			}
			continue
		}

		if matches := cueCommentRe.FindStringSubmatch(trimmed); len(matches) > 1 {
			pendingDoc = matches[1]
			continue
		}

		if matches := cuePackageRe.FindStringSubmatch(trimmed); len(matches) > 1 {
			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:      matches[1],
				Kind:      KindType,
				LineStart: lineNum,
				LineEnd:   lineNum,
				Signature: "package",
				Exported:  true,
			})
			pendingDoc = ""
			continue
		}

		if matches := cueLetRe.FindStringSubmatch(trimmed); len(matches) > 1 {
			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:       matches[1],
				Kind:       KindVariable,
				LineStart:  lineNum,
				LineEnd:    lineNum,
				DocComment: pendingDoc,
				Exported:   true,
			})
			pendingDoc = ""
			continue
		}

		if matches := cueDefinitionRe.FindStringSubmatch(line); len(matches) > 1 && !strings.HasPrefix(strings.TrimSpace(line), "//") {
			if currentDef != "" && braceCount == 0 {
				for j := range analysis.Symbols {
					if analysis.Symbols[j].Name == currentDef {
						analysis.Symbols[j].LineEnd = lineNum - 1
						break
					}
				}
			}

			name := matches[1]
			kind := KindClass
			if strings.HasPrefix(name, "#") || strings.HasPrefix(line, "#") {
				kind = KindType
				name = "#" + strings.TrimPrefix(name, "#")
			}

			analysis.Symbols = append(analysis.Symbols, Symbol{
				Name:       name,
				Kind:       kind,
				LineStart:  lineNum,
				LineEnd:    lineNum,
				DocComment: pendingDoc,
				Exported:   !strings.HasPrefix(name, "_"),
			})

			currentDef = name
			defStartLine = lineNum
			braceCount = 1
			pendingDoc = ""
			continue
		}

		if currentDef != "" && braceCount > 0 {
			if matches := cueFieldRe.FindStringSubmatch(line); len(matches) > 1 {
				fieldName := matches[1]
				if !strings.HasPrefix(fieldName, "_") {
					for j := range analysis.Symbols {
						if analysis.Symbols[j].Name == currentDef && analysis.Symbols[j].LineStart == defStartLine {
							analysis.Symbols[j].Children = append(analysis.Symbols[j].Children, Symbol{
								Name:      fieldName,
								Kind:      KindProperty,
								LineStart: lineNum,
								LineEnd:   lineNum,
								Exported:  true,
							})
							break
						}
					}
				}
			}

			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				for j := range analysis.Symbols {
					if analysis.Symbols[j].Name == currentDef && analysis.Symbols[j].LineStart == defStartLine {
						analysis.Symbols[j].LineEnd = lineNum
						break
					}
				}
				currentDef = ""
				braceCount = 0
			}
		}

		if !strings.HasPrefix(trimmed, "//") {
			pendingDoc = ""
		}
	}
}

func (p *CUEParser) extractRelationships(lines []string, analysis *FileAnalysis) {
	inImport := false

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "import") {
			inImport = true

			if strings.Contains(line, "\"") && strings.Count(line, "\"") >= 2 {
				if matches := cueImportRe.FindStringSubmatch(strings.Split(line, "import")[1]); len(matches) > 1 {
					analysis.Relationships = append(analysis.Relationships, Relationship{
						TargetFile: matches[1],
						Kind:       RelImport,
						Line:       lineNum,
					})
				}
				inImport = false
			}
			continue
		}

		if inImport {
			if strings.Contains(trimmed, ")") {
				inImport = false
				continue
			}

			if matches := cueImportRe.FindStringSubmatch(trimmed); len(matches) > 1 {
				analysis.Relationships = append(analysis.Relationships, Relationship{
					TargetFile: matches[1],
					Kind:       RelImport,
					Line:       lineNum,
				})
			}

			if strings.Contains(trimmed, "\"") && !strings.HasPrefix(trimmed, "//") {
				start := strings.Index(trimmed, "\"")
				end := strings.LastIndex(trimmed, "\"")
				if start != end && start >= 0 {
					path := trimmed[start+1 : end]
					if path != "" {
						analysis.Relationships = append(analysis.Relationships, Relationship{
							TargetFile: path,
							Kind:       RelImport,
							Line:       lineNum,
						})
					}
				}
			}
		}
	}
}
