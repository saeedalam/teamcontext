package skeleton

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/saeedalam/teamcontext/pkg/types"
)

// TypeScript/JavaScript patterns
var (
	// Class patterns
	tsClass = regexp.MustCompile(`(?m)^(\s*)(export\s+)?(abstract\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?(?:\s+implements\s+([\w,\s]+))?\s*\{`)

	// Function/method patterns
	tsFunction = regexp.MustCompile(`(?m)^(\s*)(export\s+)?(async\s+)?function\s+(\w+)\s*(<[^>]+>)?\s*\(([^)]*)\)(?:\s*:\s*([^{]+))?\s*\{`)
	tsArrowExport = regexp.MustCompile(`(?m)^(\s*)export\s+const\s+(\w+)\s*=\s*(async\s+)?\([^)]*\)(?:\s*:\s*[^=]+)?\s*=>`)
	tsMethod = regexp.MustCompile(`(?m)^(\s*)(private\s+|public\s+|protected\s+)?(static\s+)?(async\s+)?(\w+)\s*(<[^>]+>)?\s*\(([^)]*)\)(?:\s*:\s*([^{]+))?\s*\{`)
	tsConstructor = regexp.MustCompile(`(?m)^(\s*)constructor\s*\(([^)]*)\)\s*\{`)

	// Interface/type patterns
	tsInterface = regexp.MustCompile(`(?m)^(\s*)(export\s+)?interface\s+(\w+)(?:\s+extends\s+([\w,\s]+))?\s*\{`)
	tsTypeAlias = regexp.MustCompile(`(?m)^(\s*)(export\s+)?type\s+(\w+)\s*=\s*(.+)`)

	// Enum pattern
	tsEnum = regexp.MustCompile(`(?m)^(\s*)(export\s+)?enum\s+(\w+)\s*\{`)

	// Property pattern (in class)
	tsProperty = regexp.MustCompile(`(?m)^(\s*)(private\s+|public\s+|protected\s+)?(readonly\s+)?(static\s+)?(\w+)(\?)?:\s*([^;=]+)`)

	// Decorator pattern
	tsDecorator = regexp.MustCompile(`(?m)^(\s*)@(\w+)\s*\(`)
)

var (
	goFunc  = regexp.MustCompile(`(?m)^func\s+(\((\w+)\s+\*?(\w+)\)\s+)?(\w+)\s*\(([^)]*)\)(?:\s*\(([^)]*)\)|\s*(\w+(?:\s*\*?\w+)?))?\s*\{`)
	goType  = regexp.MustCompile(`(?m)^type\s+(\w+)\s+(struct|interface)\s*\{`)
	goConst = regexp.MustCompile(`(?m)^const\s+(\w+)(?:\s+(\w+))?\s*=`)
)

// Python patterns
var (
	pyClass = regexp.MustCompile(`(?m)^(\s*)class\s+(\w+)(?:\(([^)]*)\))?\s*:`)
	pyFunc = regexp.MustCompile(`(?m)^(\s*)(async\s+)?def\s+(\w+)\s*\(([^)]*)\)(?:\s*->\s*([^:]+))?\s*:`)
	pyDecorator = regexp.MustCompile(`(?m)^(\s*)@(\w+)`)
)

// Java patterns
var (
	javaClass = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|protected\s+)?(abstract\s+)?(final\s+)?class\s+(\w+)(?:<[^>]+>)?(?:\s+extends\s+(\w+))?(?:\s+implements\s+([\w,\s]+))?\s*\{`)
	javaInterface = regexp.MustCompile(`(?m)^(\s*)(public\s+)?interface\s+(\w+)(?:<[^>]+>)?(?:\s+extends\s+([\w,\s]+))?\s*\{`)
	javaMethod = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|protected\s+)?(static\s+)?(final\s+)?(synchronized\s+)?(?:(\w+(?:<[^>]+>)?(?:\[\])?)\s+)?(\w+)\s*\(([^)]*)\)(?:\s*throws\s+[\w,\s]+)?\s*\{`)
	javaField = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|protected\s+)?(static\s+)?(final\s+)?(\w+(?:<[^>]+>)?(?:\[\])?)\s+(\w+)\s*[;=]`)
	javaAnnotation = regexp.MustCompile(`(?m)^(\s*)@(\w+)`)
	javaEnum = regexp.MustCompile(`(?m)^(\s*)(public\s+)?enum\s+(\w+)\s*\{`)
)

// C# patterns
var (
	csClass = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|protected\s+|internal\s+)?(abstract\s+|sealed\s+|static\s+)?(partial\s+)?class\s+(\w+)(?:<[^>]+>)?(?:\s*:\s*([\w,\s<>]+))?\s*\{`)
	csInterface = regexp.MustCompile(`(?m)^(\s*)(public\s+|internal\s+)?interface\s+(\w+)(?:<[^>]+>)?(?:\s*:\s*([\w,\s<>]+))?\s*\{`)
	csMethod = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|protected\s+|internal\s+)?(static\s+)?(virtual\s+|override\s+|abstract\s+)?(async\s+)?(?:(\w+(?:<[^>]+>)?(?:\[\]|\?)?)\s+)?(\w+)\s*\(([^)]*)\)\s*(?:where\s+[^{]+)?\{`)
	csProperty = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|protected\s+|internal\s+)?(static\s+)?(\w+(?:<[^>]+>)?(?:\[\]|\?)?)\s+(\w+)\s*\{\s*(get|set)`)
	csAttribute = regexp.MustCompile(`(?m)^(\s*)\[(\w+)`)
	csStruct = regexp.MustCompile(`(?m)^(\s*)(public\s+|internal\s+)?(readonly\s+)?(partial\s+)?struct\s+(\w+)(?:<[^>]+>)?\s*\{`)
	csEnum = regexp.MustCompile(`(?m)^(\s*)(public\s+|internal\s+)?enum\s+(\w+)\s*\{`)
)

// Rust patterns
var (
	rustStruct = regexp.MustCompile(`(?m)^(\s*)(pub\s+)?struct\s+(\w+)(?:<[^>]+>)?\s*[\({]`)
	rustEnum   = regexp.MustCompile(`(?m)^(\s*)(pub\s+)?enum\s+(\w+)(?:<[^>]+>)?\s*\{`)
	rustTrait  = regexp.MustCompile(`(?m)^(\s*)(pub\s+)?trait\s+(\w+)(?:<[^>]+>)?(?:\s*:\s*[\w\s+]+)?\s*\{`)
	rustFn     = regexp.MustCompile(`(?m)^(\s*)(pub\s+)?(async\s+)?fn\s+(\w+)(?:<[^>]+>)?\s*\(([^)]*)\)(?:\s*->\s*([^\{]+))?\s*\{`)
	rustType   = regexp.MustCompile(`(?m)^(\s*)(pub\s+)?type\s+(\w+)(?:<[^>]+>)?\s*=`)
	rustConst  = regexp.MustCompile(`(?m)^(\s*)(pub\s+)?const\s+(\w+)\s*:\s*([^=]+)\s*=`)
	rustMacro  = regexp.MustCompile(`(?m)^(\s*)#\[(\w+)`)
)

// C/C++ patterns
var (
	cppClass     = regexp.MustCompile(`(?m)^(\s*)class\s+(\w+)(?:\s*:\s*(public|private|protected)\s+(\w+))?\s*\{`)
	cppStruct    = regexp.MustCompile(`(?m)^(\s*)struct\s+(\w+)\s*\{`)
	cppFunction  = regexp.MustCompile(`(?m)^(\s*)(?:(static|inline|virtual|explicit)\s+)*(?:(\w+(?:\s*[*&])?(?:<[^>]+>)?)\s+)?(\w+)\s*\(([^)]*)\)(?:\s*const)?(?:\s*override)?(?:\s*=\s*0)?\s*[\{;]`)
	cppNamespace = regexp.MustCompile(`(?m)^(\s*)namespace\s+(\w+)\s*\{`)
	cppTypedef   = regexp.MustCompile(`(?m)^(\s*)typedef\s+(.+)\s+(\w+)\s*;`)
	cppEnum      = regexp.MustCompile(`(?m)^(\s*)enum(?:\s+class)?\s+(\w+)(?:\s*:\s*\w+)?\s*\{`)
)

// ParseFile extracts a code skeleton from a source file
func ParseFile(filePath string) (*types.CodeSkeleton, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	lines := strings.Split(string(content), "\n")
	lineCount := len(lines)

	skeleton := &types.CodeSkeleton{
		Path:      filePath,
		LineCount: lineCount,
	}

	switch ext {
	case ".ts", ".tsx":
		skeleton.Language = "typescript"
		parseTypeScript(string(content), skeleton)
	case ".js", ".jsx", ".mjs":
		skeleton.Language = "javascript"
		parseTypeScript(string(content), skeleton) // Same patterns work
	case ".go":
		skeleton.Language = "go"
		parseGo(string(content), skeleton)
	case ".py":
		skeleton.Language = "python"
		parsePython(string(content), skeleton)
	case ".java":
		skeleton.Language = "java"
		parseJava(string(content), skeleton)
	case ".cs":
		skeleton.Language = "csharp"
		parseCSharp(string(content), skeleton)
	case ".rs":
		skeleton.Language = "rust"
		parseRust(string(content), skeleton)
	case ".c", ".h":
		skeleton.Language = "c"
		parseCpp(string(content), skeleton)
	case ".cpp", ".cc", ".cxx", ".hpp", ".hxx":
		skeleton.Language = "cpp"
		parseCpp(string(content), skeleton)
	case ".rb":
		skeleton.Language = "ruby"
		parseRuby(string(content), skeleton)
	case ".php":
		skeleton.Language = "php"
		parsePHP(string(content), skeleton)
	case ".swift":
		skeleton.Language = "swift"
		parseSwift(string(content), skeleton)
	case ".kt", ".kts":
		skeleton.Language = "kotlin"
		parseKotlin(string(content), skeleton)
	case ".scala":
		skeleton.Language = "scala"
		parseScala(string(content), skeleton)
	default:
		skeleton.Language = "unknown"
	}

	// Calculate skeleton lines (rough estimate)
	skeleton.SkeletonLines = estimateSkeletonLines(skeleton)

	return skeleton, nil
}

func parseTypeScript(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	// Track current class for adding methods
	var currentClass *types.ClassSkeleton
	inClassBody := false
	braceCount := 0

	pendingDecorators := []string{}

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Collect decorators
		if m := tsDecorator.FindStringSubmatch(line); m != nil {
			pendingDecorators = append(pendingDecorators, "@"+m[2])
			continue
		}

		// Class definition
		if m := tsClass.FindStringSubmatch(line); m != nil {
			cls := types.ClassSkeleton{
				Name:       m[4],
				Line:       lineNo,
				Extends:    m[5],
				IsAbstract: m[3] != "",
				IsExported: m[2] != "",
			}
			if m[6] != "" {
				cls.Implements = splitAndTrim(m[6], ",")
			}
			skeleton.Classes = append(skeleton.Classes, cls)
			currentClass = &skeleton.Classes[len(skeleton.Classes)-1]
			inClassBody = true
			braceCount = 1
			pendingDecorators = nil
			continue
		}

		// Track brace depth for class body
		if inClassBody {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inClassBody = false
				currentClass = nil
			}
		}

		// Constructor (inside class)
		if currentClass != nil && tsConstructor.MatchString(line) {
			if m := tsConstructor.FindStringSubmatch(line); m != nil {
				ctor := types.FunctionSig{
					Name:   "constructor",
					Line:   lineNo,
					Params: parseParams(m[2]),
				}
				currentClass.Constructor = &ctor
			}
			continue
		}

		// Method (inside class)
		if currentClass != nil {
			if m := tsMethod.FindStringSubmatch(line); m != nil {
				if m[5] != "constructor" && m[5] != "if" && m[5] != "for" && m[5] != "while" {
					method := types.FunctionSig{
						Name:       m[5],
						Line:       lineNo,
						IsPrivate:  strings.Contains(m[2], "private"),
						IsStatic:   m[3] != "",
						IsAsync:    m[4] != "",
						Params:     parseParams(m[7]),
						ReturnType: strings.TrimSpace(m[8]),
						Decorators: pendingDecorators,
					}
					currentClass.Methods = append(currentClass.Methods, method)
					pendingDecorators = nil
				}
				continue
			}

			// Property
			if m := tsProperty.FindStringSubmatch(line); m != nil {
				if !strings.Contains(line, "(") { // Not a method
					prop := types.PropertyDef{
						Name:       m[5],
						Type:       strings.TrimSpace(m[7]),
						IsPrivate:  strings.Contains(m[2], "private"),
						IsReadonly: m[3] != "",
						IsStatic:   m[4] != "",
					}
					currentClass.Properties = append(currentClass.Properties, prop)
				}
			}
			continue
		}

		// Standalone function
		if m := tsFunction.FindStringSubmatch(line); m != nil {
			fn := types.FunctionSig{
				Name:       m[4],
				Line:       lineNo,
				IsExported: m[2] != "",
				IsAsync:    m[3] != "",
				Params:     parseParams(m[6]),
				ReturnType: strings.TrimSpace(m[7]),
				Decorators: pendingDecorators,
			}
			skeleton.Functions = append(skeleton.Functions, fn)
			pendingDecorators = nil
			continue
		}

		// Arrow function export
		if m := tsArrowExport.FindStringSubmatch(line); m != nil {
			fn := types.FunctionSig{
				Name:       m[2],
				Line:       lineNo,
				IsExported: true,
				IsAsync:    m[3] != "",
			}
			skeleton.Functions = append(skeleton.Functions, fn)
			continue
		}

		// Interface
		if m := tsInterface.FindStringSubmatch(line); m != nil {
			iface := types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "interface",
				IsExported: m[2] != "",
			}
			if m[4] != "" {
				iface.Extends = splitAndTrim(m[4], ",")
			}
			skeleton.Interfaces = append(skeleton.Interfaces, iface)
			continue
		}

		// Type alias
		if m := tsTypeAlias.FindStringSubmatch(line); m != nil {
			typedef := types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "type",
				IsExported: m[2] != "",
				RawDef:     strings.TrimSpace(m[4]),
			}
			skeleton.Types = append(skeleton.Types, typedef)
			continue
		}

		// Enum
		if m := tsEnum.FindStringSubmatch(line); m != nil {
			enumDef := types.EnumDef{
				Name:       m[3],
				Line:       lineNo,
				IsExported: m[2] != "",
			}
			skeleton.Enums = append(skeleton.Enums, enumDef)
			continue
		}
	}
}

func parseGo(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Function/method
		if m := goFunc.FindStringSubmatch(line); m != nil {
			fn := types.FunctionSig{
				Name:       m[4],
				Line:       lineNo,
				IsExported: isExportedGo(m[4]),
				Params:     parseGoParams(m[5]),
				ReturnType: strings.TrimSpace(m[6] + m[7]),
			}
			skeleton.Functions = append(skeleton.Functions, fn)
			continue
		}

		// Type struct/interface
		if m := goType.FindStringSubmatch(line); m != nil {
			td := types.TypeDef{
				Name:       m[1],
				Line:       lineNo,
				Kind:       m[2],
				IsExported: isExportedGo(m[1]),
			}
			if m[2] == "interface" {
				skeleton.Interfaces = append(skeleton.Interfaces, td)
			} else {
				skeleton.Types = append(skeleton.Types, td)
			}
			continue
		}

		// Const
		if m := goConst.FindStringSubmatch(line); m != nil {
			skeleton.Constants = append(skeleton.Constants, types.ConstDef{
				Name:       m[1],
				Line:       lineNo,
				Type:       m[2],
				IsExported: isExportedGo(m[1]),
			})
			continue
		}
	}
}

func parsePython(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	var currentClass *types.ClassSkeleton
	classIndent := -1
	pendingDecorators := []string{}

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Calculate indentation
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// Reset class context if we've dedented
		if currentClass != nil && indent <= classIndent && strings.TrimSpace(line) != "" {
			currentClass = nil
			classIndent = -1
		}

		// Decorator
		if m := pyDecorator.FindStringSubmatch(line); m != nil {
			pendingDecorators = append(pendingDecorators, "@"+m[2])
			continue
		}

		// Class
		if m := pyClass.FindStringSubmatch(line); m != nil {
			cls := types.ClassSkeleton{
				Name: m[2],
				Line: lineNo,
			}
			if m[3] != "" {
				bases := splitAndTrim(m[3], ",")
				if len(bases) > 0 {
					cls.Extends = bases[0]
				}
			}
			skeleton.Classes = append(skeleton.Classes, cls)
			currentClass = &skeleton.Classes[len(skeleton.Classes)-1]
			classIndent = indent
			pendingDecorators = nil
			continue
		}

		// Function/method
		if m := pyFunc.FindStringSubmatch(line); m != nil {
			fn := types.FunctionSig{
				Name:       m[3],
				Line:       lineNo,
				IsAsync:    m[2] != "",
				Params:     parsePythonParams(m[4]),
				ReturnType: strings.TrimSpace(m[5]),
				Decorators: pendingDecorators,
			}

			// Check if it's private (starts with _)
			fn.IsPrivate = strings.HasPrefix(fn.Name, "_")

			if currentClass != nil && indent > classIndent {
				if fn.Name == "__init__" {
					currentClass.Constructor = &fn
				} else {
					currentClass.Methods = append(currentClass.Methods, fn)
				}
			} else {
				skeleton.Functions = append(skeleton.Functions, fn)
			}
			pendingDecorators = nil
			continue
		}
	}
}

// parseJava extracts skeleton from Java files
func parseJava(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	var currentClass *types.ClassSkeleton
	braceCount := 0
	inClassBody := false
	pendingAnnotations := []string{}

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Annotation
		if m := javaAnnotation.FindStringSubmatch(line); m != nil {
			pendingAnnotations = append(pendingAnnotations, "@"+m[2])
			continue
		}

		// Interface
		if m := javaInterface.FindStringSubmatch(line); m != nil {
			iface := types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "interface",
				IsExported: strings.Contains(m[2], "public"),
			}
			if m[4] != "" {
				iface.Extends = splitAndTrim(m[4], ",")
			}
			skeleton.Interfaces = append(skeleton.Interfaces, iface)
			pendingAnnotations = nil
			continue
		}

		// Class
		if m := javaClass.FindStringSubmatch(line); m != nil {
			cls := types.ClassSkeleton{
				Name:       m[5],
				Line:       lineNo,
				Extends:    m[6],
				IsAbstract: m[3] != "",
				IsExported: strings.Contains(m[2], "public"),
			}
			if m[7] != "" {
				cls.Implements = splitAndTrim(m[7], ",")
			}
			skeleton.Classes = append(skeleton.Classes, cls)
			currentClass = &skeleton.Classes[len(skeleton.Classes)-1]
			inClassBody = true
			braceCount = 1
			pendingAnnotations = nil
			continue
		}

		// Enum
		if m := javaEnum.FindStringSubmatch(line); m != nil {
			skeleton.Enums = append(skeleton.Enums, types.EnumDef{
				Name:       m[3],
				Line:       lineNo,
				IsExported: strings.Contains(m[2], "public"),
			})
			continue
		}

		// Track braces
		if inClassBody {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inClassBody = false
				currentClass = nil
			}
		}

		// Method (inside class)
		if currentClass != nil {
			if m := javaMethod.FindStringSubmatch(line); m != nil {
				method := types.FunctionSig{
					Name:       m[7],
					Line:       lineNo,
					IsPrivate:  strings.Contains(m[2], "private"),
					IsStatic:   m[3] != "",
					Params:     parseJavaParams(m[8]),
					ReturnType: m[6],
					Decorators: pendingAnnotations,
				}
				currentClass.Methods = append(currentClass.Methods, method)
				pendingAnnotations = nil
				continue
			}

			// Field
			if m := javaField.FindStringSubmatch(line); m != nil {
				prop := types.PropertyDef{
					Name:      m[6],
					Type:      m[5],
					IsPrivate: strings.Contains(m[2], "private"),
					IsStatic:  m[3] != "",
				}
				currentClass.Properties = append(currentClass.Properties, prop)
			}
		}
	}
}

// parseCSharp extracts skeleton from C# files
func parseCSharp(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	var currentClass *types.ClassSkeleton
	braceCount := 0
	inClassBody := false
	pendingAttributes := []string{}

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Attribute
		if m := csAttribute.FindStringSubmatch(line); m != nil {
			pendingAttributes = append(pendingAttributes, "["+m[2]+"]")
			continue
		}

		// Interface
		if m := csInterface.FindStringSubmatch(line); m != nil {
			iface := types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "interface",
				IsExported: strings.Contains(m[2], "public"),
			}
			if m[4] != "" {
				iface.Extends = splitAndTrim(m[4], ",")
			}
			skeleton.Interfaces = append(skeleton.Interfaces, iface)
			pendingAttributes = nil
			continue
		}

		// Struct
		if m := csStruct.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name:       m[5],
				Line:       lineNo,
				Kind:       "struct",
				IsExported: strings.Contains(m[2], "public"),
			})
			continue
		}

		// Class
		if m := csClass.FindStringSubmatch(line); m != nil {
			cls := types.ClassSkeleton{
				Name:       m[5],
				Line:       lineNo,
				IsAbstract: strings.Contains(m[3], "abstract"),
				IsExported: strings.Contains(m[2], "public"),
			}
			if m[6] != "" {
				parts := splitAndTrim(m[6], ",")
				if len(parts) > 0 {
					cls.Extends = parts[0]
					cls.Implements = parts[1:]
				}
			}
			skeleton.Classes = append(skeleton.Classes, cls)
			currentClass = &skeleton.Classes[len(skeleton.Classes)-1]
			inClassBody = true
			braceCount = 1
			pendingAttributes = nil
			continue
		}

		// Enum
		if m := csEnum.FindStringSubmatch(line); m != nil {
			skeleton.Enums = append(skeleton.Enums, types.EnumDef{
				Name:       m[3],
				Line:       lineNo,
				IsExported: strings.Contains(m[2], "public"),
			})
			continue
		}

		// Track braces
		if inClassBody {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inClassBody = false
				currentClass = nil
			}
		}

		// Method
		if currentClass != nil {
			if m := csMethod.FindStringSubmatch(line); m != nil {
				method := types.FunctionSig{
					Name:       m[7],
					Line:       lineNo,
					IsPrivate:  strings.Contains(m[2], "private"),
					IsStatic:   m[3] != "",
					IsAsync:    m[5] != "",
					Params:     parseCSharpParams(m[8]),
					ReturnType: m[6],
					Decorators: pendingAttributes,
				}
				currentClass.Methods = append(currentClass.Methods, method)
				pendingAttributes = nil
				continue
			}

			// Property
			if m := csProperty.FindStringSubmatch(line); m != nil {
				prop := types.PropertyDef{
					Name:      m[5],
					Type:      m[4],
					IsPrivate: strings.Contains(m[2], "private"),
					IsStatic:  m[3] != "",
				}
				currentClass.Properties = append(currentClass.Properties, prop)
			}
		}
	}
}

// parseRust extracts skeleton from Rust files
func parseRust(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")
	pendingMacros := []string{}

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Macro attribute
		if m := rustMacro.FindStringSubmatch(line); m != nil {
			pendingMacros = append(pendingMacros, "#["+m[2]+"]")
			continue
		}

		// Struct
		if m := rustStruct.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "struct",
				IsExported: m[2] != "",
			})
			pendingMacros = nil
			continue
		}

		// Enum
		if m := rustEnum.FindStringSubmatch(line); m != nil {
			skeleton.Enums = append(skeleton.Enums, types.EnumDef{
				Name:       m[3],
				Line:       lineNo,
				IsExported: m[2] != "",
			})
			pendingMacros = nil
			continue
		}

		// Trait
		if m := rustTrait.FindStringSubmatch(line); m != nil {
			skeleton.Interfaces = append(skeleton.Interfaces, types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "trait",
				IsExported: m[2] != "",
			})
			pendingMacros = nil
			continue
		}

		// Function
		if m := rustFn.FindStringSubmatch(line); m != nil {
			fn := types.FunctionSig{
				Name:       m[4],
				Line:       lineNo,
				IsExported: m[2] != "",
				IsAsync:    m[3] != "",
				Params:     parseRustParams(m[5]),
				ReturnType: strings.TrimSpace(m[6]),
				Decorators: pendingMacros,
			}
			skeleton.Functions = append(skeleton.Functions, fn)
			pendingMacros = nil
			continue
		}

		// Type alias
		if m := rustType.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "type",
				IsExported: m[2] != "",
			})
			continue
		}

		// Const
		if m := rustConst.FindStringSubmatch(line); m != nil {
			skeleton.Constants = append(skeleton.Constants, types.ConstDef{
				Name:       m[3],
				Line:       lineNo,
				Type:       strings.TrimSpace(m[4]),
				IsExported: m[2] != "",
			})
		}
	}
}

// parseCpp extracts skeleton from C/C++ files
func parseCpp(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	var currentClass *types.ClassSkeleton
	braceCount := 0
	inClassBody := false

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Namespace
		if m := cppNamespace.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name: m[2],
				Line: lineNo,
				Kind: "namespace",
			})
			continue
		}

		// Class
		if m := cppClass.FindStringSubmatch(line); m != nil {
			cls := types.ClassSkeleton{
				Name:    m[2],
				Line:    lineNo,
				Extends: m[4],
			}
			skeleton.Classes = append(skeleton.Classes, cls)
			currentClass = &skeleton.Classes[len(skeleton.Classes)-1]
			inClassBody = true
			braceCount = 1
			continue
		}

		// Struct
		if m := cppStruct.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name: m[2],
				Line: lineNo,
				Kind: "struct",
			})
			continue
		}

		// Enum
		if m := cppEnum.FindStringSubmatch(line); m != nil {
			skeleton.Enums = append(skeleton.Enums, types.EnumDef{
				Name: m[2],
				Line: lineNo,
			})
			continue
		}

		// Track braces
		if inClassBody {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inClassBody = false
				currentClass = nil
			}
		}

		// Function (outside class)
		if !inClassBody {
			if m := cppFunction.FindStringSubmatch(line); m != nil {
				fn := types.FunctionSig{
					Name:       m[4],
					Line:       lineNo,
					Params:     parseCppParams(m[5]),
					ReturnType: m[3],
					IsStatic:   strings.Contains(line, "static"),
				}
				skeleton.Functions = append(skeleton.Functions, fn)
				continue
			}
		}

		// Method (inside class)
		if currentClass != nil {
			if m := cppFunction.FindStringSubmatch(line); m != nil {
				method := types.FunctionSig{
					Name:       m[4],
					Line:       lineNo,
					Params:     parseCppParams(m[5]),
					ReturnType: m[3],
					IsStatic:   strings.Contains(line, "static"),
					IsPrivate:  false, // Would need to track access specifiers
				}
				currentClass.Methods = append(currentClass.Methods, method)
			}
		}

		// Typedef
		if m := cppTypedef.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name:   m[3],
				Line:   lineNo,
				Kind:   "typedef",
				RawDef: m[2],
			})
		}
	}
}

// Ruby patterns
var (
	rbClass = regexp.MustCompile(`(?m)^(\s*)class\s+(\w+)(?:\s*<\s*(\w+))?\s*$`)
	rbModule = regexp.MustCompile(`(?m)^(\s*)module\s+(\w+)\s*$`)
	rbMethod = regexp.MustCompile(`(?m)^(\s*)def\s+(self\.)?(\w+[?!=]?)\s*(?:\(([^)]*)\))?`)
	rbAttr = regexp.MustCompile(`(?m)^(\s*)attr_(reader|writer|accessor)\s+(.+)`)
)

func parseRuby(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	var currentClass *types.ClassSkeleton
	classIndent := -1

	for lineNum, line := range lines {
		lineNo := lineNum + 1
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// Reset class context
		if currentClass != nil && indent <= classIndent && strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "end") {
			currentClass = nil
			classIndent = -1
		}

		// Module
		if m := rbModule.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name: m[2],
				Line: lineNo,
				Kind: "module",
			})
			continue
		}

		// Class
		if m := rbClass.FindStringSubmatch(line); m != nil {
			cls := types.ClassSkeleton{
				Name:    m[2],
				Line:    lineNo,
				Extends: m[3],
			}
			skeleton.Classes = append(skeleton.Classes, cls)
			currentClass = &skeleton.Classes[len(skeleton.Classes)-1]
			classIndent = indent
			continue
		}

		// Method
		if m := rbMethod.FindStringSubmatch(line); m != nil {
			fn := types.FunctionSig{
				Name:     m[3],
				Line:     lineNo,
				IsStatic: m[2] != "",
				Params:   parseRubyParams(m[4]),
			}
			if currentClass != nil && indent > classIndent {
				currentClass.Methods = append(currentClass.Methods, fn)
			} else {
				skeleton.Functions = append(skeleton.Functions, fn)
			}
			continue
		}

		// Attributes
		if currentClass != nil {
			if m := rbAttr.FindStringSubmatch(line); m != nil {
				attrs := splitAndTrim(m[3], ",")
				for _, attr := range attrs {
					attr = strings.TrimPrefix(attr, ":")
					currentClass.Properties = append(currentClass.Properties, types.PropertyDef{
						Name: attr,
					})
				}
			}
		}
	}
}

// PHP patterns
var (
	phpClass = regexp.MustCompile(`(?m)^(\s*)(abstract\s+|final\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?(?:\s+implements\s+([\w,\s\\]+))?\s*\{`)
	phpInterface = regexp.MustCompile(`(?m)^(\s*)interface\s+(\w+)(?:\s+extends\s+([\w,\s\\]+))?\s*\{`)
	phpTrait = regexp.MustCompile(`(?m)^(\s*)trait\s+(\w+)\s*\{`)
	phpMethod = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|protected\s+)?(static\s+)?function\s+(\w+)\s*\(([^)]*)\)(?:\s*:\s*\??([\w\\]+))?\s*\{`)
	phpProperty = regexp.MustCompile(`(?m)^(\s*)(public|private|protected)\s+(static\s+)?(?:\??([\w\\]+)\s+)?\$(\w+)`)
)

func parsePHP(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	var currentClass *types.ClassSkeleton
	braceCount := 0
	inClassBody := false

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Interface
		if m := phpInterface.FindStringSubmatch(line); m != nil {
			iface := types.TypeDef{
				Name: m[2],
				Line: lineNo,
				Kind: "interface",
			}
			if m[3] != "" {
				iface.Extends = splitAndTrim(m[3], ",")
			}
			skeleton.Interfaces = append(skeleton.Interfaces, iface)
			continue
		}

		// Trait
		if m := phpTrait.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name: m[2],
				Line: lineNo,
				Kind: "trait",
			})
			continue
		}

		// Class
		if m := phpClass.FindStringSubmatch(line); m != nil {
			cls := types.ClassSkeleton{
				Name:       m[3],
				Line:       lineNo,
				Extends:    m[4],
				IsAbstract: strings.Contains(m[2], "abstract"),
			}
			if m[5] != "" {
				cls.Implements = splitAndTrim(m[5], ",")
			}
			skeleton.Classes = append(skeleton.Classes, cls)
			currentClass = &skeleton.Classes[len(skeleton.Classes)-1]
			inClassBody = true
			braceCount = 1
			continue
		}

		// Track braces
		if inClassBody {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inClassBody = false
				currentClass = nil
			}
		}

		// Method
		if m := phpMethod.FindStringSubmatch(line); m != nil {
			fn := types.FunctionSig{
				Name:       m[4],
				Line:       lineNo,
				IsPrivate:  strings.Contains(m[2], "private"),
				IsStatic:   m[3] != "",
				Params:     parsePHPParams(m[5]),
				ReturnType: m[6],
			}
			if currentClass != nil && inClassBody {
				currentClass.Methods = append(currentClass.Methods, fn)
			} else {
				skeleton.Functions = append(skeleton.Functions, fn)
			}
			continue
		}

		// Property
		if currentClass != nil {
			if m := phpProperty.FindStringSubmatch(line); m != nil {
				currentClass.Properties = append(currentClass.Properties, types.PropertyDef{
					Name:      m[5],
					Type:      m[4],
					IsPrivate: m[2] == "private",
					IsStatic:  m[3] != "",
				})
			}
		}
	}
}

// Swift patterns
var (
	swiftClass = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|internal\s+|open\s+)?(final\s+)?class\s+(\w+)(?:<[^>]+>)?(?:\s*:\s*([\w,\s<>]+))?\s*\{`)
	swiftStruct = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|internal\s+)?struct\s+(\w+)(?:<[^>]+>)?(?:\s*:\s*([\w,\s<>]+))?\s*\{`)
	swiftProtocol = regexp.MustCompile(`(?m)^(\s*)(public\s+)?protocol\s+(\w+)(?:\s*:\s*([\w,\s]+))?\s*\{`)
	swiftFunc = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|internal\s+|open\s+)?(static\s+|class\s+)?func\s+(\w+)(?:<[^>]+>)?\s*\(([^)]*)\)(?:\s*->\s*([^\{]+))?\s*\{`)
	swiftVar = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|internal\s+)?(static\s+)?(var|let)\s+(\w+)\s*:\s*([^\{=]+)`)
	swiftEnum = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+)?enum\s+(\w+)(?:<[^>]+>)?(?:\s*:\s*([\w,\s]+))?\s*\{`)
)

func parseSwift(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	var currentClass *types.ClassSkeleton
	braceCount := 0
	inClassBody := false

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Protocol
		if m := swiftProtocol.FindStringSubmatch(line); m != nil {
			iface := types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "protocol",
				IsExported: strings.Contains(m[2], "public"),
			}
			skeleton.Interfaces = append(skeleton.Interfaces, iface)
			continue
		}

		// Struct
		if m := swiftStruct.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "struct",
				IsExported: strings.Contains(m[2], "public"),
			})
			continue
		}

		// Class
		if m := swiftClass.FindStringSubmatch(line); m != nil {
			cls := types.ClassSkeleton{
				Name:       m[4],
				Line:       lineNo,
				IsExported: strings.Contains(m[2], "public") || strings.Contains(m[2], "open"),
			}
			if m[5] != "" {
				parts := splitAndTrim(m[5], ",")
				if len(parts) > 0 {
					cls.Extends = parts[0]
					cls.Implements = parts[1:]
				}
			}
			skeleton.Classes = append(skeleton.Classes, cls)
			currentClass = &skeleton.Classes[len(skeleton.Classes)-1]
			inClassBody = true
			braceCount = 1
			continue
		}

		// Enum
		if m := swiftEnum.FindStringSubmatch(line); m != nil {
			skeleton.Enums = append(skeleton.Enums, types.EnumDef{
				Name:       m[3],
				Line:       lineNo,
				IsExported: strings.Contains(m[2], "public"),
			})
			continue
		}

		// Track braces
		if inClassBody {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inClassBody = false
				currentClass = nil
			}
		}

		// Function
		if m := swiftFunc.FindStringSubmatch(line); m != nil {
			fn := types.FunctionSig{
				Name:       m[4],
				Line:       lineNo,
				IsExported: strings.Contains(m[2], "public") || strings.Contains(m[2], "open"),
				IsPrivate:  strings.Contains(m[2], "private"),
				IsStatic:   m[3] != "",
				Params:     parseSwiftParams(m[5]),
				ReturnType: strings.TrimSpace(m[6]),
			}
			if currentClass != nil && inClassBody {
				currentClass.Methods = append(currentClass.Methods, fn)
			} else {
				skeleton.Functions = append(skeleton.Functions, fn)
			}
			continue
		}

		// Property
		if currentClass != nil {
			if m := swiftVar.FindStringSubmatch(line); m != nil {
				currentClass.Properties = append(currentClass.Properties, types.PropertyDef{
					Name:       m[5],
					Type:       strings.TrimSpace(m[6]),
					IsPrivate:  strings.Contains(m[2], "private"),
					IsStatic:   m[3] != "",
					IsReadonly: m[4] == "let",
				})
			}
		}
	}
}

// Kotlin patterns
var (
	ktClass = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|internal\s+|protected\s+)?(abstract\s+|open\s+|data\s+|sealed\s+)?class\s+(\w+)(?:<[^>]+>)?(?:\s*\([^)]*\))?(?:\s*:\s*([\w,\s<>()]+))?\s*\{?`)
	ktInterface = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+)?interface\s+(\w+)(?:<[^>]+>)?(?:\s*:\s*([\w,\s<>]+))?\s*\{`)
	ktObject = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+)?object\s+(\w+)(?:\s*:\s*([\w,\s<>()]+))?\s*\{`)
	ktFun = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|internal\s+|protected\s+)?(suspend\s+)?fun\s+(?:<[^>]+>\s*)?(\w+)\s*\(([^)]*)\)(?:\s*:\s*([^\{=]+))?\s*[\{=]`)
	ktVal = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+|internal\s+)?(val|var)\s+(\w+)\s*:\s*([^\{=]+)`)
	ktEnum = regexp.MustCompile(`(?m)^(\s*)(public\s+|private\s+)?enum\s+class\s+(\w+)\s*\{`)
)

func parseKotlin(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	var currentClass *types.ClassSkeleton
	braceCount := 0
	inClassBody := false

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Interface
		if m := ktInterface.FindStringSubmatch(line); m != nil {
			iface := types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "interface",
				IsExported: !strings.Contains(m[2], "private"),
			}
			skeleton.Interfaces = append(skeleton.Interfaces, iface)
			continue
		}

		// Object
		if m := ktObject.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name:       m[3],
				Line:       lineNo,
				Kind:       "object",
				IsExported: !strings.Contains(m[2], "private"),
			})
			continue
		}

		// Class
		if m := ktClass.FindStringSubmatch(line); m != nil {
			cls := types.ClassSkeleton{
				Name:       m[4],
				Line:       lineNo,
				IsAbstract: strings.Contains(m[3], "abstract"),
				IsExported: !strings.Contains(m[2], "private"),
			}
			if m[5] != "" {
				parts := splitAndTrim(m[5], ",")
				if len(parts) > 0 {
					cls.Extends = strings.TrimSpace(strings.Split(parts[0], "(")[0])
				}
			}
			skeleton.Classes = append(skeleton.Classes, cls)
			currentClass = &skeleton.Classes[len(skeleton.Classes)-1]
			if strings.Contains(line, "{") {
				inClassBody = true
				braceCount = 1
			}
			continue
		}

		// Enum
		if m := ktEnum.FindStringSubmatch(line); m != nil {
			skeleton.Enums = append(skeleton.Enums, types.EnumDef{
				Name:       m[3],
				Line:       lineNo,
				IsExported: !strings.Contains(m[2], "private"),
			})
			continue
		}

		// Track braces
		if inClassBody {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inClassBody = false
				currentClass = nil
			}
		}

		// Function
		if m := ktFun.FindStringSubmatch(line); m != nil {
			fn := types.FunctionSig{
				Name:       m[4],
				Line:       lineNo,
				IsExported: !strings.Contains(m[2], "private"),
				IsPrivate:  strings.Contains(m[2], "private"),
				IsAsync:    m[3] != "",
				Params:     parseKotlinParams(m[5]),
				ReturnType: strings.TrimSpace(m[6]),
			}
			if currentClass != nil && inClassBody {
				currentClass.Methods = append(currentClass.Methods, fn)
			} else {
				skeleton.Functions = append(skeleton.Functions, fn)
			}
			continue
		}

		// Property
		if currentClass != nil {
			if m := ktVal.FindStringSubmatch(line); m != nil {
				currentClass.Properties = append(currentClass.Properties, types.PropertyDef{
					Name:       m[4],
					Type:       strings.TrimSpace(m[5]),
					IsPrivate:  strings.Contains(m[2], "private"),
					IsReadonly: m[3] == "val",
				})
			}
		}
	}
}

// Scala patterns
var (
	scalaClass = regexp.MustCompile(`(?m)^(\s*)(abstract\s+|final\s+|sealed\s+)?class\s+(\w+)(?:\[[^\]]+\])?(?:\([^)]*\))?(?:\s+extends\s+(\w+))?`)
	scalaTrait = regexp.MustCompile(`(?m)^(\s*)(sealed\s+)?trait\s+(\w+)(?:\[[^\]]+\])?(?:\s+extends\s+([\w\s]+))?`)
	scalaObject = regexp.MustCompile(`(?m)^(\s*)object\s+(\w+)(?:\s+extends\s+(\w+))?`)
	scalaDef = regexp.MustCompile(`(?m)^(\s*)(private\s+|protected\s+)?(override\s+)?def\s+(\w+)(?:\[[^\]]+\])?\s*(?:\([^)]*\))*(?:\s*:\s*([^\{=]+))?\s*[=\{]`)
	scalaVal = regexp.MustCompile(`(?m)^(\s*)(private\s+|protected\s+)?(val|var|lazy\s+val)\s+(\w+)\s*:\s*([^\{=]+)`)
)

func parseScala(content string, skeleton *types.CodeSkeleton) {
	lines := strings.Split(content, "\n")

	var currentClass *types.ClassSkeleton
	braceCount := 0
	inClassBody := false

	for lineNum, line := range lines {
		lineNo := lineNum + 1

		// Trait
		if m := scalaTrait.FindStringSubmatch(line); m != nil {
			skeleton.Interfaces = append(skeleton.Interfaces, types.TypeDef{
				Name: m[3],
				Line: lineNo,
				Kind: "trait",
			})
			continue
		}

		// Object
		if m := scalaObject.FindStringSubmatch(line); m != nil {
			skeleton.Types = append(skeleton.Types, types.TypeDef{
				Name: m[2],
				Line: lineNo,
				Kind: "object",
			})
			continue
		}

		// Class
		if m := scalaClass.FindStringSubmatch(line); m != nil {
			cls := types.ClassSkeleton{
				Name:       m[3],
				Line:       lineNo,
				Extends:    m[4],
				IsAbstract: strings.Contains(m[2], "abstract"),
			}
			skeleton.Classes = append(skeleton.Classes, cls)
			currentClass = &skeleton.Classes[len(skeleton.Classes)-1]
			if strings.Contains(line, "{") {
				inClassBody = true
				braceCount = 1
			}
			continue
		}

		// Track braces
		if inClassBody {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inClassBody = false
				currentClass = nil
			}
		}

		// Method
		if m := scalaDef.FindStringSubmatch(line); m != nil {
			fn := types.FunctionSig{
				Name:       m[4],
				Line:       lineNo,
				IsPrivate:  strings.Contains(m[2], "private"),
				ReturnType: strings.TrimSpace(m[5]),
			}
			if currentClass != nil && inClassBody {
				currentClass.Methods = append(currentClass.Methods, fn)
			} else {
				skeleton.Functions = append(skeleton.Functions, fn)
			}
			continue
		}

		// Property
		if currentClass != nil {
			if m := scalaVal.FindStringSubmatch(line); m != nil {
				currentClass.Properties = append(currentClass.Properties, types.PropertyDef{
					Name:       m[4],
					Type:       strings.TrimSpace(m[5]),
					IsPrivate:  strings.Contains(m[2], "private"),
					IsReadonly: strings.Contains(m[3], "val"),
				})
			}
		}
	}
}

// Language-specific parameter parsers

func parseJavaParams(paramsStr string) []types.ParamDef {
	if strings.TrimSpace(paramsStr) == "" {
		return nil
	}
	var params []types.ParamDef
	parts := strings.Split(paramsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Remove annotations
		for strings.HasPrefix(p, "@") {
			if idx := strings.Index(p, " "); idx != -1 {
				p = strings.TrimSpace(p[idx+1:])
			} else {
				break
			}
		}
		fields := strings.Fields(p)
		if len(fields) >= 2 {
			params = append(params, types.ParamDef{
				Name: fields[len(fields)-1],
				Type: strings.Join(fields[:len(fields)-1], " "),
			})
		}
	}
	return params
}

func parseCSharpParams(paramsStr string) []types.ParamDef {
	return parseJavaParams(paramsStr) // Similar syntax
}

func parseRustParams(paramsStr string) []types.ParamDef {
	if strings.TrimSpace(paramsStr) == "" {
		return nil
	}
	var params []types.ParamDef
	// Remove &self, &mut self, self
	paramsStr = strings.ReplaceAll(paramsStr, "&mut self,", "")
	paramsStr = strings.ReplaceAll(paramsStr, "&self,", "")
	paramsStr = strings.ReplaceAll(paramsStr, "self,", "")
	paramsStr = strings.TrimPrefix(paramsStr, "&mut self")
	paramsStr = strings.TrimPrefix(paramsStr, "&self")
	paramsStr = strings.TrimPrefix(paramsStr, "self")

	parts := strings.Split(paramsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if idx := strings.Index(p, ":"); idx != -1 {
			params = append(params, types.ParamDef{
				Name: strings.TrimSpace(p[:idx]),
				Type: strings.TrimSpace(p[idx+1:]),
			})
		}
	}
	return params
}

func parseCppParams(paramsStr string) []types.ParamDef {
	if strings.TrimSpace(paramsStr) == "" {
		return nil
	}
	var params []types.ParamDef
	parts := strings.Split(paramsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		fields := strings.Fields(p)
		if len(fields) >= 1 {
			name := fields[len(fields)-1]
			name = strings.TrimPrefix(name, "*")
			name = strings.TrimPrefix(name, "&")
			typeStr := ""
			if len(fields) >= 2 {
				typeStr = strings.Join(fields[:len(fields)-1], " ")
			}
			params = append(params, types.ParamDef{
				Name: name,
				Type: typeStr,
			})
		}
	}
	return params
}

func parseRubyParams(paramsStr string) []types.ParamDef {
	if strings.TrimSpace(paramsStr) == "" {
		return nil
	}
	var params []types.ParamDef
	parts := strings.Split(paramsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name := p
		if idx := strings.Index(p, "="); idx != -1 {
			name = strings.TrimSpace(p[:idx])
		}
		params = append(params, types.ParamDef{Name: name})
	}
	return params
}

func parsePHPParams(paramsStr string) []types.ParamDef {
	if strings.TrimSpace(paramsStr) == "" {
		return nil
	}
	var params []types.ParamDef
	parts := strings.Split(paramsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		param := types.ParamDef{}
		// Type hint
		if idx := strings.LastIndex(p, "$"); idx > 0 {
			param.Type = strings.TrimSpace(p[:idx])
			p = p[idx:]
		}
		// Name
		if idx := strings.Index(p, "="); idx != -1 {
			param.Name = strings.TrimSpace(strings.TrimPrefix(p[:idx], "$"))
			param.Default = strings.TrimSpace(p[idx+1:])
		} else {
			param.Name = strings.TrimSpace(strings.TrimPrefix(p, "$"))
		}
		params = append(params, param)
	}
	return params
}

func parseSwiftParams(paramsStr string) []types.ParamDef {
	if strings.TrimSpace(paramsStr) == "" {
		return nil
	}
	var params []types.ParamDef
	parts := strings.Split(paramsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		param := types.ParamDef{}
		// Swift params: label name: Type or _ name: Type
		if idx := strings.Index(p, ":"); idx != -1 {
			namePart := strings.TrimSpace(p[:idx])
			param.Type = strings.TrimSpace(p[idx+1:])
			// Get the actual parameter name (second word if exists)
			names := strings.Fields(namePart)
			if len(names) >= 2 {
				param.Name = names[1]
			} else if len(names) == 1 {
				param.Name = names[0]
			}
		}
		if param.Name != "" {
			params = append(params, param)
		}
	}
	return params
}

func parseKotlinParams(paramsStr string) []types.ParamDef {
	if strings.TrimSpace(paramsStr) == "" {
		return nil
	}
	var params []types.ParamDef
	parts := strings.Split(paramsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		param := types.ParamDef{}
		if idx := strings.Index(p, ":"); idx != -1 {
			param.Name = strings.TrimSpace(p[:idx])
			typeAndDefault := strings.TrimSpace(p[idx+1:])
			if eqIdx := strings.Index(typeAndDefault, "="); eqIdx != -1 {
				param.Type = strings.TrimSpace(typeAndDefault[:eqIdx])
				param.Default = strings.TrimSpace(typeAndDefault[eqIdx+1:])
			} else {
				param.Type = typeAndDefault
			}
		} else {
			param.Name = p
		}
		params = append(params, param)
	}
	return params
}

// Helper functions

func parseParams(paramsStr string) []types.ParamDef {
	if strings.TrimSpace(paramsStr) == "" {
		return nil
	}

	var params []types.ParamDef
	// Simple split - doesn't handle nested generics perfectly
	parts := strings.Split(paramsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		param := types.ParamDef{}

		// Check for optional marker
		if strings.Contains(p, "?") {
			param.Optional = true
			p = strings.Replace(p, "?", "", 1)
		}

		// Check for default value
		if idx := strings.Index(p, "="); idx != -1 {
			param.Default = strings.TrimSpace(p[idx+1:])
			p = strings.TrimSpace(p[:idx])
		}

		// Split name and type
		if idx := strings.Index(p, ":"); idx != -1 {
			param.Name = strings.TrimSpace(p[:idx])
			param.Type = strings.TrimSpace(p[idx+1:])
		} else {
			param.Name = p
		}

		if param.Name != "" {
			params = append(params, param)
		}
	}
	return params
}

func parseGoParams(paramsStr string) []types.ParamDef {
	if strings.TrimSpace(paramsStr) == "" {
		return nil
	}

	var params []types.ParamDef
	parts := strings.Split(paramsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		parts := strings.Fields(p)
		param := types.ParamDef{}
		if len(parts) >= 1 {
			param.Name = parts[0]
		}
		if len(parts) >= 2 {
			param.Type = strings.Join(parts[1:], " ")
		}
		params = append(params, param)
	}
	return params
}

func parsePythonParams(paramsStr string) []types.ParamDef {
	if strings.TrimSpace(paramsStr) == "" {
		return nil
	}

	var params []types.ParamDef
	parts := strings.Split(paramsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "self" || p == "cls" {
			continue
		}

		param := types.ParamDef{}

		// Check for default value
		if idx := strings.Index(p, "="); idx != -1 {
			param.Default = strings.TrimSpace(p[idx+1:])
			p = strings.TrimSpace(p[:idx])
		}

		// Check for type annotation
		if idx := strings.Index(p, ":"); idx != -1 {
			param.Name = strings.TrimSpace(p[:idx])
			param.Type = strings.TrimSpace(p[idx+1:])
		} else {
			param.Name = p
		}

		if param.Name != "" {
			params = append(params, param)
		}
	}
	return params
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

func isExportedGo(name string) bool {
	if len(name) == 0 {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

func estimateSkeletonLines(skeleton *types.CodeSkeleton) int {
	lines := 0

	for _, cls := range skeleton.Classes {
		lines += 3 // class header + opening/closing brace
		lines += len(cls.Properties)
		lines += len(cls.Methods)
		if cls.Constructor != nil {
			lines++
		}
	}

	lines += len(skeleton.Functions)
	lines += len(skeleton.Interfaces) * 2
	lines += len(skeleton.Types)
	lines += len(skeleton.Enums)
	lines += len(skeleton.Constants)

	return lines
}

// FormatSkeleton creates a readable skeleton string
func FormatSkeleton(sk *types.CodeSkeleton) string {
	var sb strings.Builder

	sb.WriteString("// " + sk.Path + " - SKELETON\n")
	sb.WriteString("// Language: " + sk.Language + "\n")
	sb.WriteString("// Original: " + itoa(sk.LineCount) + " lines -> Skeleton: " + itoa(sk.SkeletonLines) + " lines\n\n")

	// Interfaces
	for _, iface := range sk.Interfaces {
		if iface.IsExported {
			sb.WriteString("export ")
		}
		sb.WriteString("interface " + iface.Name)
		if len(iface.Extends) > 0 {
			sb.WriteString(" extends " + strings.Join(iface.Extends, ", "))
		}
		sb.WriteString(" { ... }\n")
	}

	// Types
	for _, t := range sk.Types {
		if t.IsExported {
			sb.WriteString("export ")
		}
		sb.WriteString("type " + t.Name + " = " + t.RawDef + "\n")
	}

	// Enums
	for _, e := range sk.Enums {
		if e.IsExported {
			sb.WriteString("export ")
		}
		sb.WriteString("enum " + e.Name + " { ... }\n")
	}

	// Classes
	for _, cls := range sk.Classes {
		sb.WriteString("\n")
		if cls.IsExported {
			sb.WriteString("export ")
		}
		if cls.IsAbstract {
			sb.WriteString("abstract ")
		}
		sb.WriteString("class " + cls.Name)
		if cls.Extends != "" {
			sb.WriteString(" extends " + cls.Extends)
		}
		if len(cls.Implements) > 0 {
			sb.WriteString(" implements " + strings.Join(cls.Implements, ", "))
		}
		sb.WriteString(" {\n")

		// Properties
		for _, p := range cls.Properties {
			sb.WriteString("  ")
			if p.IsPrivate {
				sb.WriteString("private ")
			}
			if p.IsReadonly {
				sb.WriteString("readonly ")
			}
			if p.IsStatic {
				sb.WriteString("static ")
			}
			sb.WriteString(p.Name)
			if p.Type != "" {
				sb.WriteString(": " + p.Type)
			}
			sb.WriteString("\n")
		}

		// Constructor
		if cls.Constructor != nil {
			sb.WriteString("  constructor(" + formatParams(cls.Constructor.Params) + ")\n")
		}

		// Methods
		for _, m := range cls.Methods {
			sb.WriteString("  ")
			if m.IsPrivate {
				sb.WriteString("private ")
			}
			if m.IsStatic {
				sb.WriteString("static ")
			}
			if m.IsAsync {
				sb.WriteString("async ")
			}
			sb.WriteString(m.Name + "(" + formatParams(m.Params) + ")")
			if m.ReturnType != "" {
				sb.WriteString(": " + m.ReturnType)
			}
			sb.WriteString("\n")
		}

		sb.WriteString("}\n")
	}

	// Standalone functions
	for _, fn := range sk.Functions {
		if fn.IsExported {
			sb.WriteString("export ")
		}
		if fn.IsAsync {
			sb.WriteString("async ")
		}
		sb.WriteString("function " + fn.Name + "(" + formatParams(fn.Params) + ")")
		if fn.ReturnType != "" {
			sb.WriteString(": " + fn.ReturnType)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatParams(params []types.ParamDef) string {
	var parts []string
	for _, p := range params {
		s := p.Name
		if p.Optional {
			s += "?"
		}
		if p.Type != "" {
			s += ": " + p.Type
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", ")
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
