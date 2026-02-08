package extractor

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// APIEndpoint represents a REST API endpoint
type APIEndpoint struct {
	Method     string   `json:"method"`
	Path       string   `json:"path"`
	Handler    string   `json:"handler"`
	Controller string   `json:"controller"`
	Auth       string   `json:"auth,omitempty"`
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Params     []string `json:"params,omitempty"`
}

// KafkaHandler represents a Kafka consumer/producer
type KafkaHandler struct {
	Topic     string `json:"topic"`
	Type      string `json:"type"` // "consumer" or "producer"
	Handler   string `json:"handler"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Channel   string `json:"channel,omitempty"`
}

// APISurface represents the full API surface of an app
type APISurface struct {
	App            string         `json:"app"`
	Endpoints      []APIEndpoint  `json:"endpoints"`
	KafkaConsumers []KafkaHandler `json:"kafka_consumers,omitempty"`
	KafkaProducers []KafkaHandler `json:"kafka_producers,omitempty"`
}

// NestJS/Express patterns
var (
	// NestJS decorators
	nestControllerPattern = regexp.MustCompile(`@Controller\(['"]([^'"]*)['"]\)`)
	nestMethodPatterns    = map[string]*regexp.Regexp{
		"GET":    regexp.MustCompile(`@Get\(['"]?([^'")\s]*)?['"]?\)`),
		"POST":   regexp.MustCompile(`@Post\(['"]?([^'")\s]*)?['"]?\)`),
		"PUT":    regexp.MustCompile(`@Put\(['"]?([^'")\s]*)?['"]?\)`),
		"PATCH":  regexp.MustCompile(`@Patch\(['"]?([^'")\s]*)?['"]?\)`),
		"DELETE": regexp.MustCompile(`@Delete\(['"]?([^'")\s]*)?['"]?\)`),
	}
	nestAuthPattern   = regexp.MustCompile(`@UseGuards\(([^)]+)\)`)
	nestHandlerPattern = regexp.MustCompile(`(?m)^\s*(?:async\s+)?(\w+)\s*\(`)

	// Express patterns
	expressRouterPattern = regexp.MustCompile(`router\.(get|post|put|patch|delete)\s*\(\s*['"]([^'"]+)['"]`)
	expressAppPattern    = regexp.MustCompile(`app\.(get|post|put|patch|delete)\s*\(\s*['"]([^'"]+)['"]`)

	// Kafka patterns (NestJS microservices)
	kafkaMessagePattern = regexp.MustCompile(`@MessagePattern\(['"]([^'"]+)['"]\)`)
	kafkaEventPattern   = regexp.MustCompile(`@EventPattern\(['"]([^'"]+)['"]\)`)
	kafkaClientSend     = regexp.MustCompile(`client\.send\(['"]([^'"]+)['"]`)
	kafkaClientEmit     = regexp.MustCompile(`client\.emit\(['"]([^'"]+)['"]`)

	// Go patterns (gin, echo, chi)
	goGinPattern  = regexp.MustCompile(`(?:r|router|g|group)\.(GET|POST|PUT|PATCH|DELETE)\s*\(\s*"([^"]+)"`)
	goEchoPattern = regexp.MustCompile(`(?:e|echo|g|group)\.(GET|POST|PUT|PATCH|DELETE)\s*\(\s*"([^"]+)"`)

	// Python patterns (Flask, FastAPI, Django)
	flaskRoutePattern   = regexp.MustCompile(`@(?:app|bp|blueprint)\.(route|get|post|put|patch|delete)\s*\(\s*['"]([^'"]+)['"]`)
	fastapiRoutePattern = regexp.MustCompile(`@(?:app|router)\.(get|post|put|patch|delete)\s*\(\s*['"]([^'"]+)['"]`)
	djangoUrlPattern    = regexp.MustCompile(`path\s*\(\s*['"]([^'"]+)['"]`)

	// Java patterns (Spring Boot)
	springMappingPattern = regexp.MustCompile(`@(GetMapping|PostMapping|PutMapping|PatchMapping|DeleteMapping|RequestMapping)\s*\(\s*(?:value\s*=\s*)?['"]?([^'")\s,]+)['"]?`)
	springControllerPattern = regexp.MustCompile(`@(?:Rest)?Controller\s*(?:\(\s*['"]([^'"]*)['"]\s*\))?`)

	// C# patterns (ASP.NET)
	aspnetRoutePattern = regexp.MustCompile(`\[Http(Get|Post|Put|Patch|Delete)\s*\(\s*['"]?([^'")\]]*)?['"]?\s*\)\]`)
	aspnetControllerRoute = regexp.MustCompile(`\[Route\s*\(\s*['"]([^'"]+)['"]\s*\)\]`)
)

// ExtractAPISurface extracts API endpoints from a directory
func ExtractAPISurface(dirPath string, appName string) (*APISurface, error) {
	surface := &APISurface{
		App:            appName,
		Endpoints:      []APIEndpoint{},
		KafkaConsumers: []KafkaHandler{},
		KafkaProducers: []KafkaHandler{},
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			// Skip common non-source directories
			name := info.Name()
			if name == "node_modules" || name == "dist" || name == ".git" ||
				name == "vendor" || name == "target" || name == "bin" || name == "obj" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		text := string(content)

		// Extract based on file type
		switch ext {
		case ".ts", ".js":
			extractTSEndpoints(text, path, surface)
			extractKafkaHandlers(text, path, surface)
		case ".go":
			extractGoEndpoints(text, path, surface)
		case ".py":
			extractPythonEndpoints(text, path, surface)
		case ".java":
			extractJavaEndpoints(text, path, surface)
		case ".cs":
			extractCSharpEndpoints(text, path, surface)
		}

		return nil
	})

	return surface, err
}

// ExtractAPISurfaceFromFile extracts from a single file
func ExtractAPISurfaceFromFile(filePath string) (*APISurface, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	surface := &APISurface{
		App:            filepath.Base(filepath.Dir(filePath)),
		Endpoints:      []APIEndpoint{},
		KafkaConsumers: []KafkaHandler{},
		KafkaProducers: []KafkaHandler{},
	}

	text := string(content)
	ext := strings.ToLower(filepath.Ext(filePath))

	if ext == ".go" {
		extractGoEndpoints(text, filePath, surface)
	} else {
		extractTSEndpoints(text, filePath, surface)
		extractKafkaHandlers(text, filePath, surface)
	}

	return surface, nil
}

func extractTSEndpoints(content string, filePath string, surface *APISurface) {
	lines := strings.Split(content, "\n")

	// Find controller base path
	basePath := ""
	controllerName := ""
	if m := nestControllerPattern.FindStringSubmatch(content); m != nil {
		basePath = "/" + strings.Trim(m[1], "/")
	}

	// Extract controller name from filename
	base := filepath.Base(filePath)
	if strings.Contains(base, ".controller.") {
		controllerName = strings.Split(base, ".controller.")[0]
		// Capitalize each word (replacement for deprecated strings.Title)
		words := strings.Split(strings.ReplaceAll(controllerName, "-", " "), " ")
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
			}
		}
		controllerName = strings.Join(words, "") + "Controller"
	}

	// Track auth guards at class level
	classAuth := ""
	if m := nestAuthPattern.FindStringSubmatch(content); m != nil {
		classAuth = m[1]
	}

	// Find endpoints
	for i, line := range lines {
		for method, pattern := range nestMethodPatterns {
			if m := pattern.FindStringSubmatch(line); m != nil {
				path := basePath
				if len(m) > 1 && m[1] != "" {
					subPath := strings.Trim(m[1], "/")
					if subPath != "" {
						path = basePath + "/" + subPath
					}
				}

				// Find handler name (next function after decorator)
				handlerName := ""
				for j := i + 1; j < len(lines) && j < i+5; j++ {
					if hm := nestHandlerPattern.FindStringSubmatch(lines[j]); hm != nil {
						handlerName = hm[1]
						break
					}
				}

				// Check for method-level auth
				auth := classAuth
				if i > 0 {
					for j := i - 1; j >= 0 && j > i-5; j-- {
						if am := nestAuthPattern.FindStringSubmatch(lines[j]); am != nil {
							auth = am[1]
							break
						}
					}
				}

				endpoint := APIEndpoint{
					Method:     method,
					Path:       path,
					Handler:    handlerName,
					Controller: controllerName,
					Auth:       auth,
					File:       filePath,
					Line:       i + 1,
				}

				// Extract path params
				paramPattern := regexp.MustCompile(`:(\w+)`)
				if params := paramPattern.FindAllStringSubmatch(path, -1); params != nil {
					for _, p := range params {
						endpoint.Params = append(endpoint.Params, p[1])
					}
				}

				surface.Endpoints = append(surface.Endpoints, endpoint)
			}
		}
	}

	// Express routes
	for _, pattern := range []*regexp.Regexp{expressRouterPattern, expressAppPattern} {
		matches := pattern.FindAllStringSubmatchIndex(content, -1)
		for _, match := range matches {
			if len(match) >= 6 {
				method := strings.ToUpper(content[match[2]:match[3]])
				path := content[match[4]:match[5]]
				line := strings.Count(content[:match[0]], "\n") + 1

				surface.Endpoints = append(surface.Endpoints, APIEndpoint{
					Method: method,
					Path:   path,
					File:   filePath,
					Line:   line,
				})
			}
		}
	}
}

func extractGoEndpoints(content string, filePath string, surface *APISurface) {
	for _, pattern := range []*regexp.Regexp{goGinPattern, goEchoPattern} {
		matches := pattern.FindAllStringSubmatchIndex(content, -1)
		for _, match := range matches {
			if len(match) >= 6 {
				method := content[match[2]:match[3]]
				path := content[match[4]:match[5]]
				line := strings.Count(content[:match[0]], "\n") + 1

				surface.Endpoints = append(surface.Endpoints, APIEndpoint{
					Method: method,
					Path:   path,
					File:   filePath,
					Line:   line,
				})
			}
		}
	}
}

func extractKafkaHandlers(content string, filePath string, surface *APISurface) {
	lines := strings.Split(content, "\n")

	// Find consumers (@MessagePattern, @EventPattern)
	for i, line := range lines {
		for _, pattern := range []*regexp.Regexp{kafkaMessagePattern, kafkaEventPattern} {
			if m := pattern.FindStringSubmatch(line); m != nil {
				topic := m[1]

				// Find handler name
				handlerName := ""
				for j := i + 1; j < len(lines) && j < i+5; j++ {
					if hm := nestHandlerPattern.FindStringSubmatch(lines[j]); hm != nil {
						handlerName = hm[1]
						break
					}
				}

				surface.KafkaConsumers = append(surface.KafkaConsumers, KafkaHandler{
					Topic:   topic,
					Type:    "consumer",
					Handler: handlerName,
					File:    filePath,
					Line:    i + 1,
				})
			}
		}
	}

	// Find producers (client.send, client.emit)
	for _, pattern := range []*regexp.Regexp{kafkaClientSend, kafkaClientEmit} {
		matches := pattern.FindAllStringSubmatchIndex(content, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				topic := content[match[2]:match[3]]
				line := strings.Count(content[:match[0]], "\n") + 1

				surface.KafkaProducers = append(surface.KafkaProducers, KafkaHandler{
					Topic: topic,
					Type:  "producer",
					File:  filePath,
					Line:  line,
				})
			}
		}
	}
}

// Python endpoint extraction (Flask, FastAPI, Django)
func extractPythonEndpoints(content string, filePath string, surface *APISurface) {
	lines := strings.Split(content, "\n")

	// Flask and FastAPI patterns
	for i, line := range lines {
		// Flask: @app.route('/path') or @app.get('/path')
		if m := flaskRoutePattern.FindStringSubmatch(line); m != nil {
			method := strings.ToUpper(m[1])
			if method == "ROUTE" {
				method = "GET" // Default for @route
			}
			path := m[2]

			// Find handler name
			handlerName := ""
			for j := i + 1; j < len(lines) && j < i+3; j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "def ") {
					parts := strings.Fields(lines[j])
					if len(parts) >= 2 {
						handlerName = strings.TrimSuffix(parts[1], "(")
						handlerName = strings.Split(handlerName, "(")[0]
					}
					break
				}
			}

			surface.Endpoints = append(surface.Endpoints, APIEndpoint{
				Method:  method,
				Path:    path,
				Handler: handlerName,
				File:    filePath,
				Line:    i + 1,
			})
		}

		// FastAPI: @app.get('/path') or @router.post('/path')
		if m := fastapiRoutePattern.FindStringSubmatch(line); m != nil {
			method := strings.ToUpper(m[1])
			path := m[2]

			handlerName := ""
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				trimmed := strings.TrimSpace(lines[j])
				if strings.HasPrefix(trimmed, "async def ") || strings.HasPrefix(trimmed, "def ") {
					parts := strings.Fields(trimmed)
					idx := 1
					if parts[0] == "async" {
						idx = 2
					}
					if len(parts) > idx {
						handlerName = strings.Split(parts[idx], "(")[0]
					}
					break
				}
			}

			surface.Endpoints = append(surface.Endpoints, APIEndpoint{
				Method:  method,
				Path:    path,
				Handler: handlerName,
				File:    filePath,
				Line:    i + 1,
			})
		}
	}

	// Django URL patterns
	matches := djangoUrlPattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			path := content[match[2]:match[3]]
			line := strings.Count(content[:match[0]], "\n") + 1

			surface.Endpoints = append(surface.Endpoints, APIEndpoint{
				Method: "ANY", // Django path() doesn't specify method
				Path:   "/" + strings.Trim(path, "/"),
				File:   filePath,
				Line:   line,
			})
		}
	}
}

// Java endpoint extraction (Spring Boot)
func extractJavaEndpoints(content string, filePath string, surface *APISurface) {
	lines := strings.Split(content, "\n")

	// Find controller base path
	basePath := ""
	if m := springControllerPattern.FindStringSubmatch(content); len(m) > 1 {
		basePath = m[1]
	}

	// Also check for @RequestMapping at class level
	classRequestMapping := regexp.MustCompile(`@RequestMapping\s*\(\s*(?:value\s*=\s*)?['"]([^'"]+)['"]`)
	if m := classRequestMapping.FindStringSubmatch(content); m != nil {
		basePath = m[1]
	}

	// Find endpoints
	for i, line := range lines {
		if m := springMappingPattern.FindStringSubmatch(line); m != nil {
			annotation := m[1]
			path := m[2]

			// Convert annotation to HTTP method
			method := "GET"
			switch annotation {
			case "PostMapping":
				method = "POST"
			case "PutMapping":
				method = "PUT"
			case "PatchMapping":
				method = "PATCH"
			case "DeleteMapping":
				method = "DELETE"
			case "RequestMapping":
				// Check for method attribute
				if strings.Contains(line, "POST") {
					method = "POST"
				} else if strings.Contains(line, "PUT") {
					method = "PUT"
				}
			}

			fullPath := basePath
			if path != "" {
				fullPath = strings.TrimSuffix(basePath, "/") + "/" + strings.TrimPrefix(path, "/")
			}

			// Find method name
			handlerName := ""
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				trimmed := strings.TrimSpace(lines[j])
				if strings.Contains(trimmed, "(") && !strings.HasPrefix(trimmed, "@") && !strings.HasPrefix(trimmed, "//") {
					// Extract method name
					methodPattern := regexp.MustCompile(`(?:public|private|protected)?\s*\w+\s+(\w+)\s*\(`)
					if mm := methodPattern.FindStringSubmatch(trimmed); mm != nil {
						handlerName = mm[1]
					}
					break
				}
			}

			surface.Endpoints = append(surface.Endpoints, APIEndpoint{
				Method:  method,
				Path:    fullPath,
				Handler: handlerName,
				File:    filePath,
				Line:    i + 1,
			})
		}
	}
}

// C# endpoint extraction (ASP.NET)
func extractCSharpEndpoints(content string, filePath string, surface *APISurface) {
	lines := strings.Split(content, "\n")

	// Find controller base route
	basePath := ""
	if m := aspnetControllerRoute.FindStringSubmatch(content); m != nil {
		basePath = m[1]
		// Handle [controller] placeholder
		if strings.Contains(basePath, "[controller]") {
			// Extract controller name from filename
			base := filepath.Base(filePath)
			controllerName := strings.TrimSuffix(base, "Controller.cs")
			controllerName = strings.TrimSuffix(controllerName, ".cs")
			basePath = strings.Replace(basePath, "[controller]", strings.ToLower(controllerName), 1)
		}
	}

	// Find endpoints
	for i, line := range lines {
		if m := aspnetRoutePattern.FindStringSubmatch(line); m != nil {
			method := strings.ToUpper(m[1])
			path := m[2]

			fullPath := basePath
			if path != "" {
				fullPath = strings.TrimSuffix(basePath, "/") + "/" + strings.TrimPrefix(path, "/")
			}
			if fullPath == "" {
				fullPath = basePath
			}

			// Find method name
			handlerName := ""
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				trimmed := strings.TrimSpace(lines[j])
				if strings.Contains(trimmed, "(") && !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "//") {
					methodPattern := regexp.MustCompile(`(?:public|private|protected|async)?\s*(?:async\s+)?(?:Task<)?[\w<>]+\)?\s+(\w+)\s*\(`)
					if mm := methodPattern.FindStringSubmatch(trimmed); mm != nil {
						handlerName = mm[1]
					}
					break
				}
			}

			surface.Endpoints = append(surface.Endpoints, APIEndpoint{
				Method:  method,
				Path:    "/" + strings.Trim(fullPath, "/"),
				Handler: handlerName,
				File:    filePath,
				Line:    i + 1,
			})
		}
	}
}
