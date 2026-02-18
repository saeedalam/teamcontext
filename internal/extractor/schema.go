package extractor

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SchemaModel represents a database model
type SchemaModel struct {
	Name       string        `json:"name"`
	Fields     []SchemaField `json:"fields"`
	Relations  []Relation    `json:"relations,omitempty"`
	Attributes []string      `json:"attributes,omitempty"`
	File       string        `json:"file"`
	Line       int           `json:"line"`
}

// SchemaField represents a model field
type SchemaField struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	IsOptional bool     `json:"is_optional,omitempty"`
	IsArray    bool     `json:"is_array,omitempty"`
	Default    string   `json:"default,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
}

// Relation represents a relationship between models
type Relation struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"` // "one-to-one", "one-to-many", "many-to-many"
	Model      string   `json:"model"`
	Fields     []string `json:"fields,omitempty"`
	References []string `json:"references,omitempty"`
}

// SchemaInfo holds all extracted schema information
type SchemaInfo struct {
	Models []SchemaModel `json:"models"`
	Enums  []SchemaEnum  `json:"enums,omitempty"`
}

// SchemaEnum represents an enum definition
type SchemaEnum struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
	File   string   `json:"file"`
	Line   int      `json:"line"`
}

// Prisma patterns
var (
	prismaModelPattern = regexp.MustCompile(`(?m)^model\s+(\w+)\s*\{`)
	prismaEnumPattern  = regexp.MustCompile(`(?m)^enum\s+(\w+)\s*\{`)
	prismaFieldPattern = regexp.MustCompile(`^\s*(\w+)\s+(\w+)(\[\])?\??(.*)$`)
	prismaRelationPattern = regexp.MustCompile(`@relation\(([^)]+)\)`)
)

// ExtractSchemaModels extracts database models from Prisma schema files
func ExtractSchemaModels(path string) (*SchemaInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	schema := &SchemaInfo{
		Models: []SchemaModel{},
		Enums:  []SchemaEnum{},
	}

	// Directories to skip during schema search
	skipDirs := map[string]bool{
		"node_modules": true,
		".git":         true,
		"dist":         true,
		".nx":          true,
		"coverage":     true,
		"__pycache__":  true,
		"vendor":       true,
		".cache":       true,
	}

	if info.IsDir() {
		// Walk directory looking for schema files
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				// Skip common non-source directories
				if skipDirs[info.Name()] {
					return filepath.SkipDir
				}
				return nil
			}

			base := filepath.Base(filePath)
			if base == "schema.prisma" || strings.HasSuffix(base, ".prisma") {
				fileSchema, err := extractPrismaSchema(filePath)
				if err == nil {
					schema.Models = append(schema.Models, fileSchema.Models...)
					schema.Enums = append(schema.Enums, fileSchema.Enums...)
				}
			}
			return nil
		})
		return schema, err
	}

	// Single file
	return extractPrismaSchema(path)
}

func extractPrismaSchema(filePath string) (*SchemaInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	schema := &SchemaInfo{
		Models: []SchemaModel{},
		Enums:  []SchemaEnum{},
	}

	text := string(content)

	// Extract models
	modelMatches := prismaModelPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range modelMatches {
		modelName := text[match[2]:match[3]]
		startLine := strings.Count(text[:match[0]], "\n")

		// Find model body
		braceCount := 0
		modelStart := match[1]
		modelEnd := len(text)

		for i := match[0]; i < len(text); i++ {
			if text[i] == '{' {
				braceCount++
			} else if text[i] == '}' {
				braceCount--
				if braceCount == 0 {
					modelEnd = i
					break
				}
			}
		}

		modelBody := text[modelStart:modelEnd]

		model := SchemaModel{
			Name:   modelName,
			Fields: []SchemaField{},
			File:   filePath,
			Line:   startLine + 1,
		}

		// Parse fields
		bodyLines := strings.Split(modelBody, "\n")
		for _, line := range bodyLines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "@@") {
				continue
			}

			if m := prismaFieldPattern.FindStringSubmatch(line); m != nil {
				field := SchemaField{
					Name:       m[1],
					Type:       m[2],
					IsArray:    m[3] == "[]",
					IsOptional: strings.Contains(line, "?"),
				}

				// Extract attributes
				if strings.Contains(line, "@") {
					attrPattern := regexp.MustCompile(`@\w+(?:\([^)]*\))?`)
					attrs := attrPattern.FindAllString(line, -1)
					field.Attributes = attrs
				}

				// Extract default value
				if strings.Contains(line, "@default(") {
					defaultPattern := regexp.MustCompile(`@default\(([^)]+)\)`)
					if dm := defaultPattern.FindStringSubmatch(line); dm != nil {
						field.Default = dm[1]
					}
				}

				// Check if it's a relation
				if rm := prismaRelationPattern.FindStringSubmatch(line); rm != nil {
					relation := parseRelation(m[1], m[2], rm[1])
					if relation != nil {
						model.Relations = append(model.Relations, *relation)
					}
				}

				model.Fields = append(model.Fields, field)
			}
		}

		schema.Models = append(schema.Models, model)
	}

	// Extract enums
	enumMatches := prismaEnumPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range enumMatches {
		enumName := text[match[2]:match[3]]
		startLine := strings.Count(text[:match[0]], "\n")

		// Find enum body
		braceCount := 0
		enumEnd := len(text)
		for i := match[0]; i < len(text); i++ {
			if text[i] == '{' {
				braceCount++
			} else if text[i] == '}' {
				braceCount--
				if braceCount == 0 {
					enumEnd = i
					break
				}
			}
		}

		enumBody := text[match[1]:enumEnd]

		enum := SchemaEnum{
			Name:   enumName,
			Values: []string{},
			File:   filePath,
			Line:   startLine + 1,
		}

		// Parse values
		for _, line := range strings.Split(enumBody, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "//") {
				enum.Values = append(enum.Values, line)
			}
		}

		schema.Enums = append(schema.Enums, enum)
	}

	return schema, nil
}

func parseRelation(fieldName, fieldType, relationStr string) *Relation {
	relation := &Relation{
		Name:  fieldName,
		Model: fieldType,
	}

	// Determine relation type based on field
	if strings.HasSuffix(fieldType, "[]") {
		relation.Type = "one-to-many"
	} else {
		relation.Type = "one-to-one"
	}

	// Parse fields and references
	fieldsPattern := regexp.MustCompile(`fields:\s*\[([^\]]+)\]`)
	refsPattern := regexp.MustCompile(`references:\s*\[([^\]]+)\]`)

	if m := fieldsPattern.FindStringSubmatch(relationStr); m != nil {
		relation.Fields = parseArrayItems(m[1])
	}
	if m := refsPattern.FindStringSubmatch(relationStr); m != nil {
		relation.References = parseArrayItems(m[1])
	}

	return relation
}

func parseArrayItems(s string) []string {
	var items []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
