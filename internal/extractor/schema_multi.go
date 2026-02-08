package extractor

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Multi-language schema/model extraction

// Go patterns (GORM, sqlx)
var (
	goStructPattern = regexp.MustCompile(`(?m)^type\s+(\w+)\s+struct\s*\{`)
	goFieldPattern  = regexp.MustCompile(`^\s*(\w+)\s+(\S+).*?` + "`" + `([^` + "`" + `]+)` + "`")
	gormTagPattern = regexp.MustCompile(`gorm:"([^"]+)"`)
)

// Python patterns (SQLAlchemy, Django)
var (
	sqlalchemyModelPattern = regexp.MustCompile(`class\s+(\w+)\s*\([^)]*(?:Base|Model|db\.Model)[^)]*\)`)
	sqlalchemyColumnPattern = regexp.MustCompile(`(\w+)\s*=\s*(?:Column|db\.Column)\s*\(\s*(\w+)`)
	djangoModelPattern     = regexp.MustCompile(`class\s+(\w+)\s*\(\s*models\.Model\s*\)`)
	djangoFieldPattern     = regexp.MustCompile(`(\w+)\s*=\s*models\.(\w+Field)`)
)

// Java patterns (JPA/Hibernate)
var (
	jpaEntityPattern = regexp.MustCompile(`@Entity`)
	jpaTablePattern = regexp.MustCompile(`@Table\s*\(\s*name\s*=\s*"(\w+)"`)
	jpaClassPattern = regexp.MustCompile(`(?:public\s+)?class\s+(\w+)`)
	jpaFieldPattern = regexp.MustCompile(`(?:private|public|protected)\s+(\w+)\s+(\w+)\s*;`)
)

// TypeORM patterns
var (
	typeormEntityPattern = regexp.MustCompile(`@Entity\s*\(\s*(?:['"](\w+)['"])?\s*\)`)
	typeormColumnPattern = regexp.MustCompile(`@Column\s*\([^)]*\)\s*(\w+)\s*[?:]?\s*:\s*(\w+)`)
)

// ExtractMultiLangSchema extracts schema/models from multiple languages
func ExtractMultiLangSchema(path string) (*SchemaInfo, error) {
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
		"node_modules": true, ".git": true, "vendor": true,
		"dist": true, "target": true, "__pycache__": true,
		".nx": true, ".cache": true, "coverage": true,
	}

	if info.IsDir() {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				if info != nil && info.IsDir() && skipDirs[info.Name()] {
					return filepath.SkipDir
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(filePath))
			base := filepath.Base(filePath)

			// Prisma
			if base == "schema.prisma" || strings.HasSuffix(base, ".prisma") {
				if fileSchema, err := extractPrismaSchema(filePath); err == nil {
					schema.Models = append(schema.Models, fileSchema.Models...)
					schema.Enums = append(schema.Enums, fileSchema.Enums...)
				}
				return nil
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil
			}
			text := string(content)

			switch ext {
			case ".go":
				extractGoModels(text, filePath, schema)
			case ".py":
				extractPythonModels(text, filePath, schema)
			case ".java":
				extractJavaModels(text, filePath, schema)
			case ".ts":
				extractTypeORMModels(text, filePath, schema)
			}

			return nil
		})
		return schema, err
	}

	// Single file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(content)
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".prisma":
		return extractPrismaSchema(path)
	case ".go":
		extractGoModels(text, path, schema)
	case ".py":
		extractPythonModels(text, path, schema)
	case ".java":
		extractJavaModels(text, path, schema)
	case ".ts":
		extractTypeORMModels(text, path, schema)
	}

	return schema, nil
}

func extractGoModels(content string, filePath string, schema *SchemaInfo) {
	// Find structs with gorm or db tags
	structMatches := goStructPattern.FindAllStringSubmatchIndex(content, -1)

	for _, match := range structMatches {
		structName := content[match[2]:match[3]]
		startLine := strings.Count(content[:match[0]], "\n")

		// Find struct body
		braceCount := 0
		structEnd := len(content)
		for i := match[0]; i < len(content); i++ {
			if content[i] == '{' {
				braceCount++
			} else if content[i] == '}' {
				braceCount--
				if braceCount == 0 {
					structEnd = i
					break
				}
			}
		}

		structBody := content[match[1]:structEnd]

		// Check if it has database-related tags
		if !strings.Contains(structBody, `gorm:"`) && !strings.Contains(structBody, `db:"`) {
			continue
		}

		model := SchemaModel{
			Name:   structName,
			Fields: []SchemaField{},
			File:   filePath,
			Line:   startLine + 1,
		}

		// Parse fields
		for _, line := range strings.Split(structBody, "\n") {
			if m := goFieldPattern.FindStringSubmatch(line); m != nil {
				field := SchemaField{
					Name: m[1],
					Type: m[2],
				}

				// Extract gorm attributes
				if gm := gormTagPattern.FindStringSubmatch(m[3]); gm != nil {
					field.Attributes = []string{"gorm:" + gm[1]}
				}

				model.Fields = append(model.Fields, field)
			}
		}

		if len(model.Fields) > 0 {
			schema.Models = append(schema.Models, model)
		}
	}
}

func extractPythonModels(content string, filePath string, schema *SchemaInfo) {
	lines := strings.Split(content, "\n")

	// SQLAlchemy models
	for i, line := range lines {
		if m := sqlalchemyModelPattern.FindStringSubmatch(line); m != nil {
			model := SchemaModel{
				Name:   m[1],
				Fields: []SchemaField{},
				File:   filePath,
				Line:   i + 1,
			}

			// Parse fields in following lines
			for j := i + 1; j < len(lines) && j < i+50; j++ {
				fieldLine := lines[j]
				if strings.TrimSpace(fieldLine) == "" || strings.HasPrefix(strings.TrimSpace(fieldLine), "class ") {
					break
				}

				if fm := sqlalchemyColumnPattern.FindStringSubmatch(fieldLine); fm != nil {
					model.Fields = append(model.Fields, SchemaField{
						Name: fm[1],
						Type: fm[2],
					})
				}
			}

			if len(model.Fields) > 0 {
				schema.Models = append(schema.Models, model)
			}
		}

		// Django models
		if m := djangoModelPattern.FindStringSubmatch(line); m != nil {
			model := SchemaModel{
				Name:   m[1],
				Fields: []SchemaField{},
				File:   filePath,
				Line:   i + 1,
			}

			for j := i + 1; j < len(lines) && j < i+50; j++ {
				fieldLine := lines[j]
				if strings.TrimSpace(fieldLine) == "" {
					continue
				}
				if strings.HasPrefix(strings.TrimSpace(fieldLine), "class ") {
					break
				}

				if fm := djangoFieldPattern.FindStringSubmatch(fieldLine); fm != nil {
					model.Fields = append(model.Fields, SchemaField{
						Name: fm[1],
						Type: fm[2],
					})
				}
			}

			if len(model.Fields) > 0 {
				schema.Models = append(schema.Models, model)
			}
		}
	}
}

func extractJavaModels(content string, filePath string, schema *SchemaInfo) {
	// Check if file has @Entity annotation
	if !jpaEntityPattern.MatchString(content) {
		return
	}

	lines := strings.Split(content, "\n")

	// Find class name
	className := ""
	classLine := 0
	for i, line := range lines {
		if m := jpaClassPattern.FindStringSubmatch(line); m != nil {
			className = m[1]
			classLine = i + 1
			break
		}
	}

	if className == "" {
		return
	}

	model := SchemaModel{
		Name:   className,
		Fields: []SchemaField{},
		File:   filePath,
		Line:   classLine,
	}

	// Check for table name
	if m := jpaTablePattern.FindStringSubmatch(content); m != nil {
		model.Attributes = []string{"table:" + m[1]}
	}

	// Find fields
	for _, line := range lines {
		if m := jpaFieldPattern.FindStringSubmatch(line); m != nil {
			// Skip common non-model fields
			if m[2] == "serialVersionUID" {
				continue
			}
			model.Fields = append(model.Fields, SchemaField{
				Name: m[2],
				Type: m[1],
			})
		}
	}

	if len(model.Fields) > 0 {
		schema.Models = append(schema.Models, model)
	}
}

func extractTypeORMModels(content string, filePath string, schema *SchemaInfo) {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		if m := typeormEntityPattern.FindStringSubmatch(line); m != nil {
			// Find class name in following lines
			className := ""
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				classPattern := regexp.MustCompile(`(?:export\s+)?class\s+(\w+)`)
				if cm := classPattern.FindStringSubmatch(lines[j]); cm != nil {
					className = cm[1]
					break
				}
			}

			if className == "" {
				continue
			}

			model := SchemaModel{
				Name:   className,
				Fields: []SchemaField{},
				File:   filePath,
				Line:   i + 1,
			}

			if m[1] != "" {
				model.Attributes = []string{"table:" + m[1]}
			}

			// Find columns
			for j := i; j < len(lines); j++ {
				if cm := typeormColumnPattern.FindStringSubmatch(lines[j]); cm != nil {
					model.Fields = append(model.Fields, SchemaField{
						Name: cm[1],
						Type: cm[2],
					})
				}

				// Stop at next class or end of file
				if j > i+5 && strings.Contains(lines[j], "class ") {
					break
				}
			}

			if len(model.Fields) > 0 {
				schema.Models = append(schema.Models, model)
			}
		}
	}
}
