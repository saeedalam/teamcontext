package blueprint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saeedalam/teamcontext/internal/storage"
	"github.com/saeedalam/teamcontext/pkg/types"
)

func setupTestProject(t *testing.T) (string, string, *storage.JSONStore, func()) {
	t.Helper()
	
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "blueprint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Create .teamcontext directory
	tcDir := filepath.Join(tmpDir, ".teamcontext")
	dirs := []string{
		filepath.Join(tcDir, "knowledge"),
		filepath.Join(tcDir, "index"),
		filepath.Join(tcDir, "features"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}
	
	store := storage.NewJSONStore(tcDir)
	
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}
	
	return tmpDir, tcDir, store, cleanup
}

func createNestJSProject(t *testing.T, projectDir string) {
	t.Helper()
	
	// Create package.json with NestJS
	packageJSON := `{
  "name": "test-nestjs-app",
  "dependencies": {
    "@nestjs/common": "^10.0.0",
    "@nestjs/core": "^10.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
	
	// Create app structure
	appDir := filepath.Join(projectDir, "src", "app", "users")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}
	
	// Create example controller
	controllerCode := `
import { Controller, Get, Post, Body, Param } from '@nestjs/common';
import { UsersService } from './users.service';

@Controller('users')
export class UsersController {
  constructor(private readonly usersService: UsersService) {}

  @Get()
  findAll() {
    return this.usersService.findAll();
  }

  @Post()
  create(@Body() createUserDto: CreateUserDto) {
    return this.usersService.create(createUserDto);
  }
}
`
	if err := os.WriteFile(filepath.Join(appDir, "users.controller.ts"), []byte(controllerCode), 0644); err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	
	// Create example service
	serviceCode := `
import { Injectable } from '@nestjs/common';

@Injectable()
export class UsersService {
  findAll() {
    return [];
  }

  create(dto: any) {
    return dto;
  }
}
`
	if err := os.WriteFile(filepath.Join(appDir, "users.service.ts"), []byte(serviceCode), 0644); err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
}

// =============================================================================
// BLUEPRINT GENERATION TESTS
// =============================================================================

func TestGenerateEndpointBlueprint(t *testing.T) {
	projectDir, tcDir, store, cleanup := setupTestProject(t)
	defer cleanup()
	
	createNestJSProject(t, projectDir)
	
	// Add a decision to the store
	decision := &types.Decision{
		Content: "Use class-validator for DTOs",
		Reason:  "Consistent validation across endpoints",
		Status:  "active",
		Tags:    []string{"validation"},
	}
	if err := store.AddDecision(decision); err != nil {
		t.Fatalf("Failed to add decision: %v", err)
	}
	
	generator := NewGenerator(projectDir, tcDir, store)
	blueprint, err := generator.Generate(TaskAddEndpoint, "test-app", "users")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	// Verify blueprint fields
	if blueprint.TaskType != TaskAddEndpoint {
		t.Errorf("Expected TaskType '%s', got '%s'", TaskAddEndpoint, blueprint.TaskType)
	}
	
	if blueprint.App != "test-app" {
		t.Errorf("Expected App 'test-app', got '%s'", blueprint.App)
	}
	
	// Should have checklist
	if len(blueprint.Checklist) == 0 {
		t.Error("Checklist should not be empty")
	}
	
	// Should have description
	if blueprint.Description == "" {
		t.Error("Description should not be empty")
	}
	
	// Confidence should be set
	if blueprint.Confidence == 0 {
		t.Error("Confidence should be set")
	}
	
	t.Logf("Blueprint: TaskType=%s, Confidence=%.2f, Checklist=%d items, Decisions=%d",
		blueprint.TaskType, blueprint.Confidence, len(blueprint.Checklist), len(blueprint.Decisions))
}

func TestGenerateFeatureBlueprint(t *testing.T) {
	projectDir, tcDir, store, cleanup := setupTestProject(t)
	defer cleanup()
	
	createNestJSProject(t, projectDir)
	
	generator := NewGenerator(projectDir, tcDir, store)
	blueprint, err := generator.Generate(TaskAddFeature, "test-app", "payments")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	if blueprint.TaskType != TaskAddFeature {
		t.Errorf("Expected TaskType '%s', got '%s'", TaskAddFeature, blueprint.TaskType)
	}
	
	// Feature blueprint should have checklist
	if len(blueprint.Checklist) == 0 {
		t.Error("Feature blueprint should have checklist")
	}
	
	t.Logf("Feature blueprint: %d checklist items", len(blueprint.Checklist))
}

func TestGenerateTestBlueprint(t *testing.T) {
	projectDir, tcDir, store, cleanup := setupTestProject(t)
	defer cleanup()
	
	createNestJSProject(t, projectDir)
	
	generator := NewGenerator(projectDir, tcDir, store)
	blueprint, err := generator.Generate(TaskAddTest, "test-app", "users")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	if blueprint.TaskType != TaskAddTest {
		t.Errorf("Expected TaskType '%s', got '%s'", TaskAddTest, blueprint.TaskType)
	}
	
	t.Logf("Test blueprint: %d checklist items", len(blueprint.Checklist))
}

func TestGenerateFixBugBlueprint(t *testing.T) {
	projectDir, tcDir, store, cleanup := setupTestProject(t)
	defer cleanup()
	
	createNestJSProject(t, projectDir)
	
	generator := NewGenerator(projectDir, tcDir, store)
	blueprint, err := generator.Generate(TaskFixBug, "test-app", "")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	if blueprint.TaskType != TaskFixBug {
		t.Errorf("Expected TaskType '%s', got '%s'", TaskFixBug, blueprint.TaskType)
	}
	
	// fix-bug without path should have lower confidence
	t.Logf("Fix-bug blueprint: confidence=%.2f", blueprint.Confidence)
}

func TestGenerateRefactorBlueprint(t *testing.T) {
	projectDir, tcDir, store, cleanup := setupTestProject(t)
	defer cleanup()
	
	createNestJSProject(t, projectDir)
	
	generator := NewGenerator(projectDir, tcDir, store)
	blueprint, err := generator.Generate(TaskRefactor, "test-app", "")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	if blueprint.TaskType != TaskRefactor {
		t.Errorf("Expected TaskType '%s', got '%s'", TaskRefactor, blueprint.TaskType)
	}
}

// =============================================================================
// BLUEPRINT WITH KNOWLEDGE TESTS
// =============================================================================

func TestBlueprintIncludesDecisions(t *testing.T) {
	projectDir, tcDir, store, cleanup := setupTestProject(t)
	defer cleanup()
	
	createNestJSProject(t, projectDir)
	
	// Add relevant decision
	decision := &types.Decision{
		Content:      "All endpoints must use JWT authentication",
		Reason:       "Security requirement",
		Status:       "active",
		RelatedFiles: []string{"src/auth/jwt.guard.ts"},
		Tags:         []string{"auth", "security", "endpoint"},
	}
	if err := store.AddDecision(decision); err != nil {
		t.Fatalf("Failed to add decision: %v", err)
	}
	
	generator := NewGenerator(projectDir, tcDir, store)
	blueprint, err := generator.Generate(TaskAddEndpoint, "test-app", "orders")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	// Log what decisions were included
	t.Logf("Blueprint has %d decisions", len(blueprint.Decisions))
	for _, d := range blueprint.Decisions {
		t.Logf("  - %s", d.Title)
	}
}

func TestBlueprintIncludesWarnings(t *testing.T) {
	projectDir, tcDir, store, cleanup := setupTestProject(t)
	defer cleanup()
	
	createNestJSProject(t, projectDir)
	
	// Add relevant warning
	warning := &types.Warning{
		Content:  "Don't use raw SQL queries in controllers",
		Reason:   "Security risk - SQL injection",
		Severity: "critical",
		Tags:     []string{"security", "endpoint"},
	}
	if err := store.AddWarning(warning); err != nil {
		t.Fatalf("Failed to add warning: %v", err)
	}
	
	generator := NewGenerator(projectDir, tcDir, store)
	blueprint, err := generator.Generate(TaskAddEndpoint, "test-app", "products")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	// Log what warnings were included
	t.Logf("Blueprint has %d warnings", len(blueprint.Warnings))
	for _, w := range blueprint.Warnings {
		t.Logf("  - %s (severity: %s)", w.Title, w.Severity)
	}
}

// =============================================================================
// EDGE CASES
// =============================================================================

func TestBlueprintEmptyProject(t *testing.T) {
	projectDir, tcDir, store, cleanup := setupTestProject(t)
	defer cleanup()
	
	// Don't create any project files
	
	generator := NewGenerator(projectDir, tcDir, store)
	blueprint, err := generator.Generate(TaskAddEndpoint, "test-app", "")
	if err != nil {
		t.Fatalf("Generate should handle empty project: %v", err)
	}
	
	// Should return a blueprint even for empty project
	if blueprint == nil {
		t.Fatal("Should return blueprint for empty project")
	}
	
	t.Logf("Empty project blueprint: confidence=%.2f, source=%s", blueprint.Confidence, blueprint.Source)
}

func TestBlueprintMultipleTaskTypes(t *testing.T) {
	projectDir, tcDir, store, cleanup := setupTestProject(t)
	defer cleanup()
	
	createNestJSProject(t, projectDir)
	
	generator := NewGenerator(projectDir, tcDir, store)
	
	taskTypes := []TaskType{
		TaskAddEndpoint,
		TaskAddFeature,
		TaskAddService,
		TaskFixBug,
		TaskRefactor,
		TaskAddTest,
	}
	
	for _, taskType := range taskTypes {
		blueprint, err := generator.Generate(taskType, "test-app", "")
		if err != nil {
			t.Errorf("Generate failed for task type %s: %v", taskType, err)
			continue
		}
		
		if blueprint.TaskType != taskType {
			t.Errorf("Expected TaskType '%s', got '%s'", taskType, blueprint.TaskType)
		}
		
		t.Logf("Task %s: confidence=%.2f, checklist=%d items", 
			taskType, blueprint.Confidence, len(blueprint.Checklist))
	}
}
