package typeregistry

import (
	"os"
	"regexp"
	"strings"

	"github.com/saeedalam/teamcontext/pkg/types"
)

// TypeScript type extraction patterns
var (
	// Interface with body
	interfacePattern = regexp.MustCompile(`(?ms)^(\s*)(export\s+)?interface\s+(\w+)(?:<[^>]+>)?(?:\s+extends\s+([\w,\s<>]+))?\s*\{([^}]*)\}`)

	// Type alias (single line and multi-line)
	typeAliasPattern = regexp.MustCompile(`(?m)^(\s*)(export\s+)?type\s+(\w+)(?:<[^>]+>)?\s*=\s*(.+)$`)

	// Enum with body
	enumPattern = regexp.MustCompile(`(?ms)^(\s*)(export\s+)?enum\s+(\w+)\s*\{([^}]*)\}`)

	// Property inside interface/type
	propertyPattern = regexp.MustCompile(`(?m)^\s*(?:readonly\s+)?(\w+)(\?)?:\s*([^;,]+)`)
)

// ExtractTypes extracts all type definitions from a TypeScript file
func ExtractTypes(filePath string) ([]types.TypeDef, []types.EnumDef, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, err
	}

	var typeDefs []types.TypeDef
	var enumDefs []types.EnumDef

	text := string(content)
	lines := strings.Split(text, "\n")

	// Extract interfaces
	matches := interfacePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) < 12 {
			continue
		}

		fullMatch := text[match[0]:match[1]]
		isExported := match[4] != match[5] // export keyword present
		name := text[match[6]:match[7]]
		extendsStr := ""
		if match[8] != -1 && match[9] != -1 {
			extendsStr = text[match[8]:match[9]]
		}
		body := ""
		if match[10] != -1 && match[11] != -1 {
			body = text[match[10]:match[11]]
		}

		// Find line number
		lineNo := findLineNumber(lines, match[0])

		td := types.TypeDef{
			Name:       name,
			Line:       lineNo,
			Kind:       "interface",
			IsExported: isExported,
			RawDef:     strings.TrimSpace(fullMatch),
		}

		if extendsStr != "" {
			td.Extends = splitAndTrim(extendsStr, ",")
		}

		// Parse properties from body
		td.Properties = parseInterfaceProperties(body)

		typeDefs = append(typeDefs, td)
	}

	// Extract type aliases
	for lineNum, line := range lines {
		if m := typeAliasPattern.FindStringSubmatch(line); m != nil {
			td := types.TypeDef{
				Name:       m[3],
				Line:       lineNum + 1,
				Kind:       "type",
				IsExported: m[2] != "",
				RawDef:     strings.TrimSpace(m[4]),
			}

			// For multi-line types, try to capture more
			if strings.HasSuffix(strings.TrimSpace(m[4]), "{") || strings.HasSuffix(strings.TrimSpace(m[4]), "(") {
				td.RawDef = captureMultilineType(lines, lineNum)
			}

			typeDefs = append(typeDefs, td)
		}
	}

	// Extract enums
	enumMatches := enumPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range enumMatches {
		if len(match) < 10 {
			continue
		}

		isExported := match[4] != match[5]
		name := text[match[6]:match[7]]
		body := ""
		if match[8] != -1 && match[9] != -1 {
			body = text[match[8]:match[9]]
		}

		lineNo := findLineNumber(lines, match[0])

		ed := types.EnumDef{
			Name:       name,
			Line:       lineNo,
			IsExported: isExported,
			Members:    parseEnumMembers(body),
		}

		enumDefs = append(enumDefs, ed)
	}

	return typeDefs, enumDefs, nil
}

// ExtractTypesFromContent extracts types from raw content string
func ExtractTypesFromContent(content string, filePath string) ([]types.TypeDef, []types.EnumDef) {
	var typeDefs []types.TypeDef
	var enumDefs []types.EnumDef

	lines := strings.Split(content, "\n")

	// Extract interfaces
	matches := interfacePattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		if len(match) < 12 {
			continue
		}

		fullMatch := content[match[0]:match[1]]
		isExported := match[4] != match[5]
		name := content[match[6]:match[7]]
		extendsStr := ""
		if match[8] != -1 && match[9] != -1 {
			extendsStr = content[match[8]:match[9]]
		}
		body := ""
		if match[10] != -1 && match[11] != -1 {
			body = content[match[10]:match[11]]
		}

		lineNo := findLineNumber(lines, match[0])

		td := types.TypeDef{
			Name:       name,
			Line:       lineNo,
			Kind:       "interface",
			IsExported: isExported,
			RawDef:     strings.TrimSpace(fullMatch),
			Properties: parseInterfaceProperties(body),
		}

		if extendsStr != "" {
			td.Extends = splitAndTrim(extendsStr, ",")
		}

		typeDefs = append(typeDefs, td)
	}

	// Extract type aliases
	for lineNum, line := range lines {
		if m := typeAliasPattern.FindStringSubmatch(line); m != nil {
			td := types.TypeDef{
				Name:       m[3],
				Line:       lineNum + 1,
				Kind:       "type",
				IsExported: m[2] != "",
				RawDef:     strings.TrimSpace(m[4]),
			}

			if strings.HasSuffix(strings.TrimSpace(m[4]), "{") || strings.HasSuffix(strings.TrimSpace(m[4]), "(") {
				td.RawDef = captureMultilineType(lines, lineNum)
			}

			typeDefs = append(typeDefs, td)
		}
	}

	// Extract enums
	enumMatches := enumPattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range enumMatches {
		if len(match) < 10 {
			continue
		}

		isExported := match[4] != match[5]
		name := content[match[6]:match[7]]
		body := ""
		if match[8] != -1 && match[9] != -1 {
			body = content[match[8]:match[9]]
		}

		lineNo := findLineNumber(lines, match[0])

		ed := types.EnumDef{
			Name:       name,
			Line:       lineNo,
			IsExported: isExported,
			Members:    parseEnumMembers(body),
		}

		enumDefs = append(enumDefs, ed)
	}

	return typeDefs, enumDefs
}

// FormatTypeDefs creates a clean string representation of types
func FormatTypeDefs(typeDefs []types.TypeDef, enumDefs []types.EnumDef) string {
	var sb strings.Builder

	for _, td := range typeDefs {
		if td.IsExported {
			sb.WriteString("export ")
		}

		if td.Kind == "interface" {
			sb.WriteString("interface " + td.Name)
			if len(td.Extends) > 0 {
				sb.WriteString(" extends " + strings.Join(td.Extends, ", "))
			}
			sb.WriteString(" {\n")
			for _, p := range td.Properties {
				sb.WriteString("  " + p.Name)
				if p.Type != "" {
					sb.WriteString(": " + p.Type)
				}
				sb.WriteString(";\n")
			}
			sb.WriteString("}\n\n")
		} else {
			sb.WriteString("type " + td.Name + " = " + td.RawDef + "\n\n")
		}
	}

	for _, ed := range enumDefs {
		if ed.IsExported {
			sb.WriteString("export ")
		}
		sb.WriteString("enum " + ed.Name + " {\n")
		for _, m := range ed.Members {
			sb.WriteString("  " + m + ",\n")
		}
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

// Helper functions

func parseInterfaceProperties(body string) []types.PropertyDef {
	var props []types.PropertyDef

	matches := propertyPattern.FindAllStringSubmatch(body, -1)
	for _, m := range matches {
		prop := types.PropertyDef{
			Name: m[1],
			Type: strings.TrimSpace(m[3]),
		}
		props = append(props, prop)
	}

	return props
}

func parseEnumMembers(body string) []string {
	var members []string

	// Split by comma or newline
	parts := regexp.MustCompile(`[,\n]+`).Split(body, -1)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// Extract just the member name (before = if present)
		if idx := strings.Index(p, "="); idx != -1 {
			p = strings.TrimSpace(p[:idx])
		}
		if p != "" {
			members = append(members, p)
		}
	}

	return members
}

func findLineNumber(lines []string, charIndex int) int {
	count := 0
	for i, line := range lines {
		count += len(line) + 1 // +1 for newline
		if count > charIndex {
			return i + 1
		}
	}
	return 1
}

func captureMultilineType(lines []string, startLine int) string {
	var sb strings.Builder
	braceCount := 0
	parenCount := 0
	started := false

	for i := startLine; i < len(lines); i++ {
		line := lines[i]
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}

		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
		parenCount += strings.Count(line, "(") - strings.Count(line, ")")

		if strings.Contains(line, "{") || strings.Contains(line, "(") {
			started = true
		}

		if started && braceCount <= 0 && parenCount <= 0 {
			break
		}
	}

	return sb.String()
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	var result []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}
