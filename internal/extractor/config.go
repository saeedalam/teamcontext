package extractor

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ConfigVar represents an environment/config variable
type ConfigVar struct {
	Name        string `json:"name"`
	Source      string `json:"source"` // "env", "config", "secret"
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
	File        string `json:"file"`
	Line        int    `json:"line"`
}

// ConfigMap holds all extracted configuration
type ConfigMap struct {
	EnvVars     []ConfigVar `json:"env_vars"`
	ConfigFiles []string    `json:"config_files"`
}

// Patterns for config extraction
var (
	// Node.js / TypeScript
	processEnvPattern    = regexp.MustCompile(`process\.env\.(\w+)`)
	processEnvBracket    = regexp.MustCompile(`process\.env\[['"](\w+)['"]\]`)
	configGetPattern     = regexp.MustCompile(`config(?:Service)?\.get[^\(]*\(['"]([^'"]+)['"]`)

	// Go
	osGetenvPattern      = regexp.MustCompile(`os\.Getenv\(['"](\w+)['"]\)`)
	osLookupEnvPattern   = regexp.MustCompile(`os\.LookupEnv\(['"](\w+)['"]\)`)
	viperGetPattern      = regexp.MustCompile(`viper\.Get(?:String|Int|Bool|Duration)?\(['"]([^'"]+)['"]\)`)
	envStructTagPattern  = regexp.MustCompile(`env:"(\w+)"`)

	// Python
	osEnvironPattern     = regexp.MustCompile(`os\.environ(?:\.get)?\[?['"](\w+)['"]\]?`)
	osGetenvPyPattern    = regexp.MustCompile(`os\.getenv\(['"](\w+)['"]`)

	// .env file pattern
	envFilePattern       = regexp.MustCompile(`^(\w+)=(.*)$`)
)

// ExtractConfigMap extracts all config/env vars from a directory
func ExtractConfigMap(dirPath string) (*ConfigMap, error) {
	configMap := &ConfigMap{
		EnvVars:     []ConfigVar{},
		ConfigFiles: []string{},
	}

	seen := make(map[string]bool)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == "dist" || name == ".git" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		base := filepath.Base(path)
		ext := strings.ToLower(filepath.Ext(path))

		// Track config files
		if isConfigFile(base) {
			configMap.ConfigFiles = append(configMap.ConfigFiles, path)
		}

		// Parse .env files
		if strings.HasPrefix(base, ".env") {
			vars, _ := extractEnvFile(path)
			for _, v := range vars {
				if !seen[v.Name] {
					seen[v.Name] = true
					configMap.EnvVars = append(configMap.EnvVars, v)
				}
			}
			return nil
		}

		// Parse source files for env var usage
		if ext == ".ts" || ext == ".js" || ext == ".go" || ext == ".py" {
			vars, _ := extractEnvUsage(path)
			for _, v := range vars {
				if !seen[v.Name] {
					seen[v.Name] = true
					configMap.EnvVars = append(configMap.EnvVars, v)
				}
			}
		}

		return nil
	})

	return configMap, err
}

func isConfigFile(name string) bool {
	configNames := []string{
		".env", ".env.local", ".env.development", ".env.production", ".env.staging",
		"config.yaml", "config.yml", "config.json", "config.toml",
		"application.yaml", "application.yml", "application.properties",
		"settings.py", "settings.json",
	}
	for _, n := range configNames {
		if name == n || strings.HasPrefix(name, ".env") {
			return true
		}
	}
	return false
}

func extractEnvFile(filePath string) ([]ConfigVar, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var vars []ConfigVar
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if m := envFilePattern.FindStringSubmatch(line); m != nil {
			vars = append(vars, ConfigVar{
				Name:    m[1],
				Default: m[2],
				Source:  "env",
				File:    filePath,
				Line:    i + 1,
			})
		}
	}

	return vars, nil
}

func extractEnvUsage(filePath string) ([]ConfigVar, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var vars []ConfigVar
	text := string(content)
	lines := strings.Split(text, "\n")
	ext := filepath.Ext(filePath)

	// Select patterns based on file type
	var patterns []*regexp.Regexp
	switch ext {
	case ".ts", ".js":
		patterns = []*regexp.Regexp{processEnvPattern, processEnvBracket, configGetPattern}
	case ".go":
		patterns = []*regexp.Regexp{osGetenvPattern, osLookupEnvPattern, viperGetPattern}
	case ".py":
		patterns = []*regexp.Regexp{osEnvironPattern, osGetenvPyPattern}
	}

	seen := make(map[string]bool)

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				varName := text[match[2]:match[3]]
				if seen[varName] {
					continue
				}
				seen[varName] = true

				lineNum := strings.Count(text[:match[0]], "\n") + 1

				source := "env"
				if strings.Contains(pattern.String(), "config") || strings.Contains(pattern.String(), "viper") {
					source = "config"
				}

				vars = append(vars, ConfigVar{
					Name:   varName,
					Source: source,
					File:   filePath,
					Line:   lineNum,
				})
			}
		}
	}

	// Also check struct tags for Go
	if ext == ".go" {
		for i, line := range lines {
			if m := envStructTagPattern.FindStringSubmatch(line); m != nil {
				if !seen[m[1]] {
					seen[m[1]] = true
					vars = append(vars, ConfigVar{
						Name:   m[1],
						Source: "env",
						File:   filePath,
						Line:   i + 1,
					})
				}
			}
		}
	}

	return vars, nil
}
