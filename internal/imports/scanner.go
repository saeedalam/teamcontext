package imports

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/saeedalam/teamcontext/pkg/types"
)

// Language-specific import patterns
var (
	// TypeScript/JavaScript
	tsImportFrom   = regexp.MustCompile(`import\s+(?:(?:[\w*{}\s,]+)\s+from\s+)?['"]([^'"]+)['"]`)
	tsRequire      = regexp.MustCompile(`(?:require|import)\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	tsDynamicImport = regexp.MustCompile(`import\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	// Go
	goSingleImport = regexp.MustCompile(`^\s*import\s+"([^"]+)"`)
	goBlockImport  = regexp.MustCompile(`^\s*"([^"]+)"`)

	// Python
	pyImport     = regexp.MustCompile(`^\s*import\s+([\w.]+)`)
	pyFromImport = regexp.MustCompile(`^\s*from\s+([\w.]+)\s+import`)
)

// ScanFile parses imports from a source file
func ScanFile(filePath string) ([]types.ImportResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	scanner := bufio.NewScanner(f)

	var results []types.ImportResult

	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		results = scanTypeScript(scanner, filePath)
	case ".go":
		results = scanGo(scanner, filePath)
	case ".py":
		results = scanPython(scanner, filePath)
	default:
		// Try TypeScript patterns as fallback
		results = scanTypeScript(scanner, filePath)
	}

	return results, scanner.Err()
}

func scanTypeScript(scanner *bufio.Scanner, source string) []types.ImportResult {
	var results []types.ImportResult
	seen := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()

		for _, re := range []*regexp.Regexp{tsImportFrom, tsRequire, tsDynamicImport} {
			matches := re.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				if len(m) < 2 {
					continue
				}
				imported := m[1]
				if seen[imported] {
					continue
				}
				seen[imported] = true

				result := types.ImportResult{
					Source:     source,
					Imported:   imported,
					ImportType: classifyTSImport(imported),
					Raw:        strings.TrimSpace(line),
				}

				// Resolve relative imports
				if strings.HasPrefix(imported, ".") {
					result.Imported = resolveRelative(source, imported)
				}

				results = append(results, result)
			}
		}
	}
	return results
}

func scanGo(scanner *bufio.Scanner, source string) []types.ImportResult {
	var results []types.ImportResult
	seen := make(map[string]bool)
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Single-line import
		if m := goSingleImport.FindStringSubmatch(line); len(m) >= 2 {
			imp := m[1]
			if !seen[imp] {
				seen[imp] = true
				results = append(results, types.ImportResult{
					Source:     source,
					Imported:   imp,
					ImportType: classifyGoImport(imp),
					Raw:        trimmed,
				})
			}
			continue
		}

		// Block import start
		if strings.HasPrefix(trimmed, "import (") || trimmed == "import (" {
			inBlock = true
			continue
		}

		// Block import end
		if inBlock && trimmed == ")" {
			inBlock = false
			continue
		}

		// Inside block import
		if inBlock {
			if m := goBlockImport.FindStringSubmatch(line); len(m) >= 2 {
				imp := m[1]
				if !seen[imp] {
					seen[imp] = true
					results = append(results, types.ImportResult{
						Source:     source,
						Imported:   imp,
						ImportType: classifyGoImport(imp),
						Raw:        trimmed,
					})
				}
			}
		}
	}
	return results
}

func scanPython(scanner *bufio.Scanner, source string) []types.ImportResult {
	var results []types.ImportResult
	seen := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()

		// from X import ...
		if m := pyFromImport.FindStringSubmatch(line); len(m) >= 2 {
			imp := m[1]
			if !seen[imp] {
				seen[imp] = true
				result := types.ImportResult{
					Source:     source,
					Imported:   imp,
					ImportType: classifyPythonImport(imp),
					Raw:        strings.TrimSpace(line),
				}
				if strings.HasPrefix(imp, ".") {
					result.Imported = resolveRelative(source, imp)
				}
				results = append(results, result)
			}
			continue
		}

		// import X
		if m := pyImport.FindStringSubmatch(line); len(m) >= 2 {
			imp := m[1]
			if !seen[imp] {
				seen[imp] = true
				results = append(results, types.ImportResult{
					Source:     source,
					Imported:   imp,
					ImportType: classifyPythonImport(imp),
					Raw:        strings.TrimSpace(line),
				})
			}
		}
	}
	return results
}

// classifyTSImport determines the import type for TS/JS imports
func classifyTSImport(imported string) string {
	if strings.HasPrefix(imported, ".") {
		return "relative"
	}
	if strings.HasPrefix(imported, "@") || !strings.Contains(imported, "/") || strings.Contains(imported, "node_modules") {
		return "package"
	}
	return "package"
}

// classifyGoImport determines the import type for Go imports
func classifyGoImport(imported string) string {
	// Standard library packages don't contain dots
	if !strings.Contains(imported, ".") {
		return "builtin"
	}
	return "package"
}

// classifyPythonImport determines the import type for Python imports
func classifyPythonImport(imported string) string {
	if strings.HasPrefix(imported, ".") {
		return "relative"
	}
	// Common standard library modules
	stdlibs := map[string]bool{
		"os": true, "sys": true, "json": true, "re": true, "math": true,
		"datetime": true, "collections": true, "itertools": true, "functools": true,
		"typing": true, "pathlib": true, "io": true, "abc": true, "enum": true,
		"dataclasses": true, "logging": true, "unittest": true, "http": true,
		"urllib": true, "asyncio": true, "subprocess": true, "threading": true,
	}
	base := strings.Split(imported, ".")[0]
	if stdlibs[base] {
		return "builtin"
	}
	return "package"
}

// resolveRelative resolves a relative import path against the source file
func resolveRelative(source, imported string) string {
	dir := filepath.Dir(source)
	resolved := filepath.Join(dir, imported)
	resolved = filepath.Clean(resolved)
	return resolved
}
