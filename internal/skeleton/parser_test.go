package skeleton

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saeedalam/teamcontext/pkg/types"
)

func setupTestFile(t *testing.T, content string, ext string) (string, func()) {
	t.Helper()
	
	tmpDir, err := os.MkdirTemp("", "skeleton-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	filePath := filepath.Join(tmpDir, "test"+ext)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}
	
	return filePath, cleanup
}

// Helper to check if skeleton contains a class by name
func hasClass(skeleton *types.CodeSkeleton, name string) bool {
	for _, c := range skeleton.Classes {
		if c.Name == name {
			return true
		}
	}
	return false
}

// Helper to check if skeleton contains a method by name in any class
func hasMethod(skeleton *types.CodeSkeleton, methodName string) bool {
	for _, c := range skeleton.Classes {
		for _, m := range c.Methods {
			if m.Name == methodName {
				return true
			}
		}
	}
	return false
}



// Helper to check if skeleton contains an interface by name
func hasInterface(skeleton *types.CodeSkeleton, name string) bool {
	for _, i := range skeleton.Interfaces {
		if i.Name == name {
			return true
		}
	}
	return false
}

// Helper to check if skeleton contains a type by name
func hasType(skeleton *types.CodeSkeleton, name string) bool {
	for _, t := range skeleton.Types {
		if t.Name == name {
			return true
		}
	}
	return false
}

// =============================================================================
// TYPESCRIPT TESTS
// =============================================================================

func TestTypeScriptClassSkeleton(t *testing.T) {
	code := `
import { Injectable } from '@nestjs/common';

@Injectable()
export class UserService {
  constructor(private prisma: PrismaService) {}

  async findAll(): Promise<User[]> {
    const users = await this.prisma.user.findMany();
    return users;
  }

  async findById(id: string): Promise<User | null> {
    return this.prisma.user.findUnique({ where: { id } });
  }
}
`
	
	filePath, cleanup := setupTestFile(t, code, ".ts")
	defer cleanup()
	
	skeleton, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	
	if skeleton.Language != "typescript" {
		t.Errorf("Expected language 'typescript', got '%s'", skeleton.Language)
	}
	
	if !hasClass(skeleton, "UserService") {
		t.Error("Skeleton should contain class 'UserService'")
	}
	
	if !hasMethod(skeleton, "findAll") {
		t.Error("Skeleton should contain 'findAll' method")
	}
	
	t.Logf("TypeScript: %d classes, %d functions, %d interfaces", 
		len(skeleton.Classes), len(skeleton.Functions), len(skeleton.Interfaces))
}

func TestTypeScriptInterface(t *testing.T) {
	code := `
export interface User {
  id: string;
  email: string;
  name: string;
}

export type UserRole = 'admin' | 'user';
`
	
	filePath, cleanup := setupTestFile(t, code, ".ts")
	defer cleanup()
	
	skeleton, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	
	if !hasInterface(skeleton, "User") {
		t.Error("Skeleton should contain interface 'User'")
	}
	
	if !hasType(skeleton, "UserRole") {
		t.Error("Skeleton should contain type 'UserRole'")
	}
}

// =============================================================================
// GO TESTS
// =============================================================================

func TestGoStructAndMethods(t *testing.T) {
	code := `
package service

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) FindByID(ctx context.Context, id string) (*User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}
`
	
	filePath, cleanup := setupTestFile(t, code, ".go")
	defer cleanup()
	
	skeleton, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	
	if skeleton.Language != "go" {
		t.Errorf("Expected language 'go', got '%s'", skeleton.Language)
	}
	
	// Go structs might be parsed as classes or types depending on implementation
	t.Logf("Go: %d classes, %d functions, %d types", 
		len(skeleton.Classes), len(skeleton.Functions), len(skeleton.Types))
	
	// Should have at least the standalone functions
	if len(skeleton.Functions) == 0 && len(skeleton.Classes) == 0 {
		t.Error("Should have parsed some functions or classes")
	}
}

func TestGoInterface(t *testing.T) {
	code := `
package repository

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, user *User) error
}
`
	
	filePath, cleanup := setupTestFile(t, code, ".go")
	defer cleanup()
	
	skeleton, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	
	// Should have interface or type
	if len(skeleton.Interfaces) == 0 && len(skeleton.Types) == 0 && len(skeleton.Classes) == 0 {
		t.Error("Should have parsed UserRepository interface")
	}
}

// =============================================================================
// PYTHON TESTS
// =============================================================================

func TestPythonClass(t *testing.T) {
	code := `
from typing import List, Optional

class UserService:
    def __init__(self, db):
        self.db = db
    
    async def get_all(self) -> List[User]:
        return self.db.query(User).all()
    
    async def get_by_id(self, user_id: str) -> Optional[User]:
        return self.db.query(User).filter(User.id == user_id).first()
`
	
	filePath, cleanup := setupTestFile(t, code, ".py")
	defer cleanup()
	
	skeleton, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	
	if skeleton.Language != "python" {
		t.Errorf("Expected language 'python', got '%s'", skeleton.Language)
	}
	
	if !hasClass(skeleton, "UserService") {
		t.Error("Skeleton should contain class 'UserService'")
	}
	
	t.Logf("Python: %d classes, %d functions", len(skeleton.Classes), len(skeleton.Functions))
}

// =============================================================================
// RUST TESTS
// =============================================================================

func TestRustStructAndImpl(t *testing.T) {
	code := `
use std::collections::HashMap;

#[derive(Debug, Clone)]
pub struct UserService {
    users: HashMap<String, User>,
}

impl UserService {
    pub fn new() -> Self {
        Self { users: HashMap::new() }
    }

    pub fn find_by_id(&self, id: &str) -> Option<&User> {
        self.users.get(id)
    }
}

pub trait Repository {
    fn find(&self, id: &str) -> Option<User>;
}
`
	
	filePath, cleanup := setupTestFile(t, code, ".rs")
	defer cleanup()
	
	skeleton, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	
	if skeleton.Language != "rust" {
		t.Errorf("Expected language 'rust', got '%s'", skeleton.Language)
	}
	
	t.Logf("Rust: %d classes, %d functions, %d interfaces", 
		len(skeleton.Classes), len(skeleton.Functions), len(skeleton.Interfaces))
}

// =============================================================================
// JAVA TESTS
// =============================================================================

func TestJavaClass(t *testing.T) {
	code := `
package com.example.service;

import java.util.List;

@Service
public class UserService {
    private final UserRepository repository;
    
    @Autowired
    public UserService(UserRepository repository) {
        this.repository = repository;
    }
    
    public List<User> findAll() {
        return repository.findAll();
    }
}
`
	
	filePath, cleanup := setupTestFile(t, code, ".java")
	defer cleanup()
	
	skeleton, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	
	if skeleton.Language != "java" {
		t.Errorf("Expected language 'java', got '%s'", skeleton.Language)
	}
	
	if !hasClass(skeleton, "UserService") {
		t.Error("Skeleton should contain class 'UserService'")
	}
}

// =============================================================================
// EDGE CASES
// =============================================================================

func TestEmptyFile(t *testing.T) {
	filePath, cleanup := setupTestFile(t, "", ".ts")
	defer cleanup()
	
	skeleton, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed on empty file: %v", err)
	}
	
	if skeleton == nil {
		t.Error("Should return skeleton even for empty file")
	}
}

func TestNonExistentFile(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/file.ts")
	if err == nil {
		t.Error("Should return error for non-existent file")
	}
}

func TestUnsupportedExtension(t *testing.T) {
	filePath, cleanup := setupTestFile(t, "SELECT * FROM users;", ".sql")
	defer cleanup()
	
	skeleton, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	
	// Should still return a skeleton, possibly with raw content
	if skeleton == nil {
		t.Error("Should return skeleton even for unsupported extension")
	}
}

// =============================================================================
// TOKEN SAVINGS TEST
// =============================================================================

func TestSkeletonTokenSavings(t *testing.T) {
	code := `
import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';

@Injectable()
export class UsersService {
  constructor(
    @InjectRepository(User)
    private usersRepository: Repository<User>,
  ) {}

  async create(createUserDto: CreateUserDto): Promise<User> {
    const user = this.usersRepository.create(createUserDto);
    await this.usersRepository.save(user);
    return user;
  }

  async findAll(): Promise<User[]> {
    const users = await this.usersRepository.find({
      relations: ['profile', 'posts'],
    });
    return users;
  }

  async findOne(id: string): Promise<User> {
    const user = await this.usersRepository.findOne({
      where: { id },
      relations: ['profile', 'posts'],
    });
    if (!user) {
      throw new NotFoundException('User not found');
    }
    return user;
  }

  async update(id: string, updateUserDto: UpdateUserDto): Promise<User> {
    const user = await this.findOne(id);
    Object.assign(user, updateUserDto);
    return this.usersRepository.save(user);
  }

  async remove(id: string): Promise<void> {
    const user = await this.findOne(id);
    await this.usersRepository.remove(user);
  }
}
`
	
	filePath, cleanup := setupTestFile(t, code, ".ts")
	defer cleanup()
	
	skeleton, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	
	// Log the parsed content
	t.Logf("Original lines: %d, Skeleton lines: %d", skeleton.LineCount, skeleton.SkeletonLines)
	t.Logf("Classes: %d, Functions: %d, Methods in classes: %d", 
		len(skeleton.Classes), 
		len(skeleton.Functions),
		func() int {
			total := 0
			for _, c := range skeleton.Classes {
				total += len(c.Methods)
			}
			return total
		}())
	
	// Should have identified the class
	if !hasClass(skeleton, "UsersService") {
		t.Error("Should have identified UsersService class")
	}
}
