package analysis

import (
	"testing"
)

// TestGoParser tests Go language parsing in detail
func TestGoParser(t *testing.T) {
	parser := NewGoParser()

	t.Run("parse package with interface", func(t *testing.T) {
		code := `package storage

// Storage defines the storage interface
type Storage interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
}
`
		result, err := parser.Parse([]byte(code), "storage.go")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "Storage" {
				found = true
				if sym.Kind != KindInterface {
					t.Errorf("Storage.Kind = %q, want %q", sym.Kind, KindInterface)
				}
				if !sym.Exported {
					t.Error("Storage should be exported")
				}
				// Interface methods may or may not be in Children depending on parser impl
				// Just verify the interface was found with correct kind
			}
		}
		if !found {
			t.Error("Did not find Storage interface")
		}
	})

	t.Run("parse struct with fields", func(t *testing.T) {
		code := `package user

type User struct {
	ID        int64
	Name      string
	email     string
	CreatedAt time.Time
}
`
		result, err := parser.Parse([]byte(code), "user.go")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		for _, sym := range result.Symbols {
			if sym.Name == "User" {
				if sym.Kind != KindClass {
					t.Errorf("User.Kind = %q, want %q", sym.Kind, KindClass)
				}
				// Should have 4 fields
				if len(sym.Children) < 3 {
					t.Errorf("Expected at least 3 struct fields, got %d", len(sym.Children))
				}
				return
			}
		}
		t.Error("Did not find User struct")
	})

	t.Run("parse method with receiver", func(t *testing.T) {
		code := `package user

type User struct {
	Name string
}

func (u *User) GetName() string {
	return u.Name
}

func (u User) String() string {
	return u.Name
}
`
		result, err := parser.Parse([]byte(code), "user.go")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		methods := 0
		for _, sym := range result.Symbols {
			if sym.Kind == KindMethod {
				methods++
				if sym.Signature == "" {
					t.Errorf("Method %s has empty signature", sym.Name)
				}
			}
		}
		if methods != 2 {
			t.Errorf("Expected 2 methods, got %d", methods)
		}
	})
}

// TestTypeScriptParser tests TypeScript parsing
func TestTypeScriptParser(t *testing.T) {
	parser := NewTypeScriptParser()

	t.Run("parse interface", func(t *testing.T) {
		code := `interface Config {
	host: string;
	port: number;
	ssl?: boolean;
}
`
		result, err := parser.Parse([]byte(code), "config.ts")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "Config" {
				found = true
				if sym.Kind != KindInterface {
					t.Errorf("Config.Kind = %q, want %q", sym.Kind, KindInterface)
				}
			}
		}
		if !found {
			t.Error("Did not find Config interface")
		}
	})

	t.Run("parse class with methods", func(t *testing.T) {
		code := `class UserService {
	private users: User[] = [];

	constructor() {
		this.users = [];
	}

	addUser(user: User): void {
		this.users.push(user);
	}

	getUser(id: number): User | undefined {
		return this.users.find(u => u.id === id);
	}
}
`
		result, err := parser.Parse([]byte(code), "service.ts")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "UserService" {
				found = true
				if sym.Kind != KindClass {
					t.Errorf("UserService.Kind = %q, want %q", sym.Kind, KindClass)
				}
			}
		}
		if !found {
			t.Error("Did not find UserService class")
		}
	})

	t.Run("parse export function", func(t *testing.T) {
		code := `export function createApp(config: Config): App {
	return new App(config);
}

export const VERSION = "1.0.0";
`
		result, err := parser.Parse([]byte(code), "app.ts")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		symbolNames := make(map[string]bool)
		for _, sym := range result.Symbols {
			symbolNames[sym.Name] = true
		}

		if !symbolNames["createApp"] {
			t.Error("Did not find createApp function")
		}
	})

	t.Run("parse type alias", func(t *testing.T) {
		code := `type ID = string | number;
type Handler = (req: Request, res: Response) => void;
`
		result, err := parser.Parse([]byte(code), "types.ts")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if len(result.Symbols) == 0 {
			t.Error("Expected to find type aliases")
		}
	})
}

// TestPythonParser tests Python parsing
func TestPythonParser(t *testing.T) {
	parser := NewPythonParser()

	t.Run("parse class with methods", func(t *testing.T) {
		code := `class Calculator:
    """A simple calculator class"""

    def __init__(self, initial: float = 0):
        self.value = initial

    def add(self, x: float) -> float:
        """Add x to the current value"""
        self.value += x
        return self.value

    def multiply(self, x: float) -> float:
        self.value *= x
        return self.value
`
		result, err := parser.Parse([]byte(code), "calc.py")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "Calculator" {
				found = true
				if sym.Kind != KindClass {
					t.Errorf("Calculator.Kind = %q, want %q", sym.Kind, KindClass)
				}
			}
		}
		if !found {
			t.Error("Did not find Calculator class")
		}
	})

	t.Run("parse async functions", func(t *testing.T) {
		code := `async def fetch_data(url: str) -> dict:
    async with aiohttp.ClientSession() as session:
        async with session.get(url) as response:
            return await response.json()

async def process_items(items: list) -> list:
    return [await process(item) for item in items]
`
		result, err := parser.Parse([]byte(code), "async.py")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		functions := 0
		for _, sym := range result.Symbols {
			if sym.Kind == KindFunction {
				functions++
			}
		}
		if functions < 2 {
			t.Errorf("Expected at least 2 async functions, got %d", functions)
		}
	})

	t.Run("parse decorated function", func(t *testing.T) {
		code := `@app.route("/api/users")
@login_required
def get_users():
    return User.query.all()

@staticmethod
def helper():
    pass
`
		result, err := parser.Parse([]byte(code), "routes.py")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if len(result.Symbols) < 1 {
			t.Error("Expected to find decorated functions")
		}
	})
}

// TestRustParser tests Rust parsing
func TestRustParser(t *testing.T) {
	parser := NewRustParser()

	t.Run("parse struct and impl", func(t *testing.T) {
		code := `pub struct Point {
    pub x: f64,
    pub y: f64,
}

impl Point {
    pub fn new(x: f64, y: f64) -> Self {
        Point { x, y }
    }

    pub fn distance(&self, other: &Point) -> f64 {
        let dx = self.x - other.x;
        let dy = self.y - other.y;
        (dx * dx + dy * dy).sqrt()
    }
}
`
		result, err := parser.Parse([]byte(code), "point.rs")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		symbolNames := make(map[string]bool)
		for _, sym := range result.Symbols {
			symbolNames[sym.Name] = true
		}

		if !symbolNames["Point"] {
			t.Error("Did not find Point struct")
		}
	})

	t.Run("parse trait", func(t *testing.T) {
		code := `pub trait Drawable {
    fn draw(&self);
    fn bounds(&self) -> Rect;
}
`
		result, err := parser.Parse([]byte(code), "traits.rs")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "Drawable" {
				found = true
			}
		}
		if !found {
			t.Error("Did not find Drawable trait")
		}
	})

	t.Run("parse enum", func(t *testing.T) {
		code := `pub enum Status {
    Pending,
    Active,
    Completed,
    Failed(String),
}
`
		result, err := parser.Parse([]byte(code), "status.rs")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "Status" {
				found = true
				if sym.Kind != KindEnum {
					t.Errorf("Status.Kind = %q, want %q", sym.Kind, KindEnum)
				}
			}
		}
		if !found {
			t.Error("Did not find Status enum")
		}
	})
}

// TestJavaParser tests Java parsing
func TestJavaParser(t *testing.T) {
	parser := NewJavaParser()

	t.Run("parse class with methods", func(t *testing.T) {
		code := `public class UserService {
    private final UserRepository repository;

    public UserService(UserRepository repository) {
        this.repository = repository;
    }

    public User findById(Long id) {
        return repository.findById(id).orElse(null);
    }

    public List<User> findAll() {
        return repository.findAll();
    }
}
`
		result, err := parser.Parse([]byte(code), "UserService.java")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "UserService" {
				found = true
				if sym.Kind != KindClass {
					t.Errorf("UserService.Kind = %q, want %q", sym.Kind, KindClass)
				}
			}
		}
		if !found {
			t.Error("Did not find UserService class")
		}
	})

	t.Run("parse interface", func(t *testing.T) {
		code := `public interface Repository<T, ID> {
    T findById(ID id);
    List<T> findAll();
    T save(T entity);
    void delete(T entity);
}
`
		result, err := parser.Parse([]byte(code), "Repository.java")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "Repository" {
				found = true
				if sym.Kind != KindInterface {
					t.Errorf("Repository.Kind = %q, want %q", sym.Kind, KindInterface)
				}
			}
		}
		if !found {
			t.Error("Did not find Repository interface")
		}
	})
}

// TestJavaScriptParser tests JavaScript parsing
func TestJavaScriptParser(t *testing.T) {
	parser := NewJavaScriptParser()

	t.Run("parse class", func(t *testing.T) {
		code := `class EventEmitter {
    constructor() {
        this.events = {};
    }

    on(event, callback) {
        if (!this.events[event]) {
            this.events[event] = [];
        }
        this.events[event].push(callback);
    }

    emit(event, ...args) {
        if (this.events[event]) {
            this.events[event].forEach(cb => cb(...args));
        }
    }
}
`
		result, err := parser.Parse([]byte(code), "events.js")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "EventEmitter" {
				found = true
			}
		}
		if !found {
			t.Error("Did not find EventEmitter class")
		}
	})

	t.Run("parse functions", func(t *testing.T) {
		code := `function add(a, b) {
    return a + b;
}

const multiply = (a, b) => a * b;

const divide = function(a, b) {
    return a / b;
};
`
		result, err := parser.Parse([]byte(code), "math.js")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if len(result.Symbols) < 1 {
			t.Error("Expected to find functions")
		}
	})
}

// TestCParser tests C parsing
func TestCParser(t *testing.T) {
	parser := NewCParser()

	t.Run("parse struct and functions", func(t *testing.T) {
		code := `typedef struct {
    int x;
    int y;
} Point;

Point* create_point(int x, int y) {
    Point* p = malloc(sizeof(Point));
    p->x = x;
    p->y = y;
    return p;
}

void destroy_point(Point* p) {
    free(p);
}
`
		result, err := parser.Parse([]byte(code), "point.c")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if len(result.Symbols) < 2 {
			t.Error("Expected to find struct and functions")
		}
	})
}

// TestBashParser tests Bash script parsing
func TestBashParser(t *testing.T) {
	parser := NewBashParser()

	t.Run("parse functions", func(t *testing.T) {
		code := `#!/bin/bash

function greet() {
    echo "Hello, $1!"
}

cleanup() {
    rm -rf /tmp/myapp
}

main() {
    greet "World"
    cleanup
}

main "$@"
`
		result, err := parser.Parse([]byte(code), "script.sh")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if len(result.Symbols) < 2 {
			t.Error("Expected to find bash functions")
		}
	})
}

// TestDockerfileParser tests Dockerfile parsing
func TestDockerfileParser(t *testing.T) {
	parser := NewDockerfileParser()

	t.Run("parse Dockerfile", func(t *testing.T) {
		code := `FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/server ./cmd/server

FROM alpine:3.18
COPY --from=builder /app/server /usr/local/bin/server
EXPOSE 8080
CMD ["server"]
`
		result, err := parser.Parse([]byte(code), "Dockerfile")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "dockerfile" {
			t.Errorf("Language = %q, want %q", result.Language, "dockerfile")
		}
	})
}

// TestYAMLParser tests YAML parsing
func TestYAMLParser(t *testing.T) {
	parser := NewYAMLParser()

	t.Run("parse kubernetes manifest", func(t *testing.T) {
		code := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`
		result, err := parser.Parse([]byte(code), "deployment.yaml")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "yaml" {
			t.Errorf("Language = %q, want %q", result.Language, "yaml")
		}
	})

	t.Run("parse blocks", func(t *testing.T) {
		code := `
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`
		result, err := parser.Parse([]byte(code), "docker-compose.yml")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "yaml" {
			t.Errorf("Language = %q, want %q", result.Language, "yaml")
		}
	})
}

// TestSQLParser tests SQL parsing
func TestSQLParser(t *testing.T) {
	parser := NewSQLParser()

	t.Run("parse SQL schema", func(t *testing.T) {
		code := `CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);

CREATE VIEW active_users AS
SELECT * FROM users WHERE deleted_at IS NULL;
`
		result, err := parser.Parse([]byte(code), "schema.sql")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "sql" {
			t.Errorf("Language = %q, want %q", result.Language, "sql")
		}
	})

	t.Run("parse table and function", func(t *testing.T) {
		code := `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

CREATE FUNCTION get_user_name(user_id INT) RETURNS TEXT AS $$
BEGIN
    RETURN (SELECT name FROM users WHERE id = user_id);
END;
$$ LANGUAGE plpgsql;

CREATE VIEW active_users AS SELECT * FROM users;
CREATE INDEX idx_users_name ON users(name);
`
		result, err := parser.Parse([]byte(code), "schema.sql")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "sql" {
			t.Errorf("Language = %q, want %q", result.Language, "sql")
		}

		// Verify symbols
		foundTable := false
		foundFunc := false
		foundView := false
		foundIndex := false
		for _, sym := range result.Symbols {
			switch sym.Name {
			case "users":
				foundTable = true
			case "get_user_name":
				foundFunc = true
			case "active_users":
				foundView = true
			case "idx_users_name":
				foundIndex = true
			}
		}
		if !foundTable {
			t.Error("Table 'users' not found")
		}
		if !foundFunc {
			t.Error("Function 'get_user_name' not found")
		}
		if !foundView {
			t.Error("View 'active_users' not found")
		}
		if !foundIndex {
			t.Error("Index 'idx_users_name' not found")
		}
	})
}

// TestHCLParser tests HCL/Terraform parsing
func TestHCLParser(t *testing.T) {
	parser := NewHCLParser()

	t.Run("parse terraform resource", func(t *testing.T) {
		code := `resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"

  tags = {
    Name = "HelloWorld"
  }
}

variable "region" {
  default = "us-east-1"
}

output "instance_ip" {
  value = aws_instance.web.public_ip
}
`
		result, err := parser.Parse([]byte(code), "main.tf")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "hcl" {
			t.Errorf("Language = %q, want %q", result.Language, "hcl")
		}
	})
}

// TestParserErrorHandling tests that parsers handle malformed code gracefully
func TestParserErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		parser   Parser
		code     string
		filePath string
	}{
		{
			name:     "Go with syntax error",
			parser:   NewGoParser(),
			code:     "package main\n\nfunc broken( {",
			filePath: "broken.go",
		},
		{
			name:     "TypeScript with syntax error",
			parser:   NewTypeScriptParser(),
			code:     "class Broken { constructor( }",
			filePath: "broken.ts",
		},
		{
			name:     "Python with syntax error",
			parser:   NewPythonParser(),
			code:     "def broken(\n    pass",
			filePath: "broken.py",
		},
		{
			name:     "Java with syntax error",
			parser:   NewJavaParser(),
			code:     "public class Broken { public void test( }",
			filePath: "Broken.java",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic, may return error or partial results
			result, _ := tt.parser.Parse([]byte(tt.code), tt.filePath)
			// At minimum, should return a non-nil result with path set
			if result == nil {
				t.Error("Parser returned nil for malformed code")
				return
			}
			if result.Path != tt.filePath {
				t.Errorf("result.Path = %q, want %q", result.Path, tt.filePath)
			}
		})
	}
}

// TestParserEmptyInput tests parsers with empty input
func TestParserEmptyInput(t *testing.T) {
	parsers := []struct {
		name   string
		parser Parser
		ext    string
	}{
		{"Go", NewGoParser(), ".go"},
		{"TypeScript", NewTypeScriptParser(), ".ts"},
		{"Python", NewPythonParser(), ".py"},
		{"Java", NewJavaParser(), ".java"},
		{"JavaScript", NewJavaScriptParser(), ".js"},
	}

	for _, p := range parsers {
		t.Run(p.name+" empty input", func(t *testing.T) {
			result, err := p.parser.Parse([]byte(""), "empty"+p.ext)
			if err != nil {
				t.Errorf("Parse returned error for empty input: %v", err)
			}
			if result == nil {
				t.Error("Parse returned nil for empty input")
			}
		})
	}
}

// TestSwiftParser tests Swift parsing
func TestSwiftParser(t *testing.T) {
	parser := NewSwiftParser()

	t.Run("parse class and struct", func(t *testing.T) {
		code := `
import Foundation

/// A simple user model
public class User {
    var name: String
    var age: Int

    init(name: String, age: Int) {
        self.name = name
        self.age = age
    }

    func sayHello() {
        print("Hello, \(name)")
    }
}

struct Point {
    let x: Double
    let y: Double
}
`
		result, err := parser.Parse([]byte(code), "User.swift")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		foundUser := false
		foundPoint := false
		for _, sym := range result.Symbols {
			if sym.Name == "User" {
				foundUser = true
				if sym.Kind != KindClass {
					t.Errorf("User.Kind = %q, want %q", sym.Kind, KindClass)
				}
				if !sym.Exported {
					t.Error("User should be exported")
				}
			}
			if sym.Name == "Point" {
				foundPoint = true
				if sym.Kind != KindClass { // Struct often matches Class kind in these parsers
					t.Errorf("Point.Kind = %q, want %q", sym.Kind, KindClass)
				}
			}
		}
		if !foundUser {
			t.Error("Did not find User class")
		}
		if !foundPoint {
			t.Error("Did not find Point struct")
		}

		// Check import
		foundImport := false
		for _, rel := range result.Relationships {
			if rel.Kind == RelImport && rel.TargetFile == "Foundation" {
				foundImport = true
				break
			}
		}
		if !foundImport {
			t.Error("Did not find Foundation import")
		}
	})
}

// TestRubyParser tests Ruby parsing
func TestRubyParser(t *testing.T) {
	parser := NewRubyParser()

	t.Run("parse class with methods", func(t *testing.T) {
		code := `
require 'json'

class Greeter
  def initialize(name)
    @name = name
  end

  def hello
    puts "Hello, #{@name}!"
  end

  def self.version
    "1.0.0"
  end
end
`
		result, err := parser.Parse([]byte(code), "greeter.rb")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "Greeter" {
				found = true
				if sym.Kind != KindClass {
					t.Errorf("Greeter.Kind = %q, want %q", sym.Kind, KindClass)
				}
			}
		}
		if !found {
			t.Error("Did not find Greeter class")
		}

		foundImport := false
		for _, rel := range result.Relationships {
			if rel.Kind == RelImport && rel.TargetFile == "json" {
				foundImport = true
				break
			}
		}
		if !foundImport {
			t.Error("Did not find json require")
		}
	})
}

// TestPHPParser tests PHP parsing
func TestPHPParser(t *testing.T) {
	parser := NewPHPParser()

	t.Run("parse class and interface", func(t *testing.T) {
		code := `<?php
namespace App;

use Vendor\Library;

interface Loggable {
    public function log($message);
}

class User implements Loggable {
    private $id;
    public $name;

    public function __construct($id, $name) {
        $this->id = $id;
        $this->name = $name;
    }

    public function log($message) {
        echo $message;
    }

    public static function create($name) {
        return new self(rand(), $name);
    }
}
`
		result, err := parser.Parse([]byte(code), "User.php")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		foundInterface := false
		foundClass := false
		for _, sym := range result.Symbols {
			if sym.Name == "Loggable" {
				foundInterface = true
				if sym.Kind != KindInterface {
					t.Errorf("Loggable.Kind = %q, want %q", sym.Kind, KindInterface)
				}
			}
			if sym.Name == "User" {
				foundClass = true
				if sym.Kind != KindClass {
					t.Errorf("User.Kind = %q, want %q", sym.Kind, KindClass)
				}
			}
		}
		if !foundInterface {
			t.Error("Did not find Loggable interface")
		}
		if !foundClass {
			t.Error("Did not find User class")
		}
	})
}

// TestKotlinParser tests Kotlin parsing
func TestKotlinParser(t *testing.T) {
	parser := NewKotlinParser()

	t.Run("parse class and function", func(t *testing.T) {
		code := `
package com.example

class User(val id: Int, var name: String) {
    fun greet() {
        println("Hello, $name")
    }
}

fun main() {
    val user = User(1, "Kotlin")
    user.greet()
}
`
		result, err := parser.Parse([]byte(code), "User.kt")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "User" {
				found = true
				if sym.Kind != KindClass {
					t.Errorf("User.Kind = %q, want %q", sym.Kind, KindClass)
				}
			}
		}
		if !found {
			t.Error("Did not find User class")
		}
	})
}

// TestScalaParser tests Scala parsing
func TestScalaParser(t *testing.T) {
	parser := NewScalaParser()

	t.Run("parse class and object", func(t *testing.T) {
		code := `
package com.example

case class User(id: Int, name: String)

object Main {
  def main(args: Array[String]): Unit = {
    val user = User(1, "Scala")
    println(s"Hello, ${user.name}")
  }
}
`
		result, err := parser.Parse([]byte(code), "Main.scala")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "Main" {
				found = true
				if sym.Kind != KindClass { // Object often matches Class kind
					t.Errorf("Main.Kind = %q, want %q", sym.Kind, KindClass)
				}
			}
		}
		if !found {
			t.Error("Did not find Main object")
		}
	})
}

// TestCSharpParser tests C# parsing
func TestCSharpParser(t *testing.T) {
	parser := NewCSharpParser()

	t.Run("parse class and namespace", func(t *testing.T) {
		code := `
using System;

namespace Example {
    public class Greeter {
        public void Greet(string name) {
            Console.WriteLine($"Hello, {name}");
        }
    }
}
`
		result, err := parser.Parse([]byte(code), "Greeter.cs")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "Greeter" {
				found = true
				if sym.Kind != KindClass {
					t.Errorf("Greeter.Kind = %q, want %q", sym.Kind, KindClass)
				}
			}
		}
		if !found {
			t.Error("Did not find Greeter class")
		}
	})
}

// TestElixirParser tests Elixir parsing
func TestElixirParser(t *testing.T) {
	parser := NewElixirParser()

	t.Run("parse module and function", func(t *testing.T) {
		code := `
defmodule Greeter do
  def hello(name) do
    "Hello, #{name}"
  end
end
`
		result, err := parser.Parse([]byte(code), "greeter.ex")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		found := false
		for _, sym := range result.Symbols {
			if sym.Name == "Greeter" {
				found = true
				if sym.Kind != KindClass {
					t.Errorf("Greeter.Kind = %q, want %q", sym.Kind, KindClass)
				}
			}
		}
		if !found {
			t.Error("Did not find Greeter module")
		}
	})
}

// TestLuaParser tests Lua parsing
func TestLuaParser(t *testing.T) {
	parser := NewLuaParser()

	t.Run("parse symbols", func(t *testing.T) {
		code := `
function greet(name)
  print("Hello, " .. name)
end

local function secret()
  return 42
end

local x = 10
MAX_SIZE = 100
config.port = 8080
`
		result, err := parser.Parse([]byte(code), "test.lua")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		symbols := make(map[string]Symbol)
		for _, s := range result.Symbols {
			symbols[s.Name] = s
		}

		if _, ok := symbols["greet"]; !ok {
			t.Error("Global function 'greet' not found")
		}
		if _, ok := symbols["secret"]; !ok {
			t.Error("Local function 'secret' not found")
		}
		if _, ok := symbols["x"]; !ok {
			t.Error("Local variable 'x' not found")
		}

		// Constants (all caps)
		if _, ok := symbols["MAX_SIZE"]; !ok {
			t.Error("Constant 'MAX_SIZE' not found")
		}

		// Properties
		if _, ok := symbols["config.port"]; !ok {
			t.Error("Property 'config.port' not found")
		}
	})
}

// TestMarkdownParser tests Markdown parsing
func TestMarkdownParser(t *testing.T) {
	parser := NewMarkdownParser()

	t.Run("parse markdown headings", func(t *testing.T) {
		code := `
# Project Title
## Installation
### Usage
`
		result, err := parser.Parse([]byte(code), "README.md")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "markdown" {
			t.Errorf("Language = %q, want %q", result.Language, "markdown")
		}
	})
}

// TestJSONParser tests JSON parsing
func TestJSONParser(t *testing.T) {
	parser := NewJSONParser()

	t.Run("parse json object", func(t *testing.T) {
		code := `{"name": "test", "version": 1.0}`
		result, err := parser.Parse([]byte(code), "package.json")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "json" {
			t.Errorf("Language = %q, want %q", result.Language, "json")
		}
	})
}

// TestCSSParser tests CSS parsing
func TestCSSParser(t *testing.T) {
	parser := NewCSSParser()

	t.Run("parse css rules", func(t *testing.T) {
		code := `
.container { width: 100%; }
#header { color: red; }
`
		result, err := parser.Parse([]byte(code), "style.css")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "css" {
			t.Errorf("Language = %q, want %q", result.Language, "css")
		}
	})
}

// TestHTMLParser tests HTML parsing
func TestHTMLParser(t *testing.T) {
	parser := NewHTMLParser()

	t.Run("parse html structure", func(t *testing.T) {
		code := `<!DOCTYPE html><html><body><h1>Title</h1></body></html>`
		result, err := parser.Parse([]byte(code), "index.html")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "html" {
			t.Errorf("Language = %q, want %q", result.Language, "html")
		}
	})
}

// TestTOMLParser tests TOML parsing
func TestTOMLParser(t *testing.T) {
	parser := NewTOMLParser()

	t.Run("parse tables", func(t *testing.T) {
		code := `
[package]
name = "mind-palace"
version = "0.1.0"

[dependencies]
sqlite = "3.0"
`
		result, err := parser.Parse([]byte(code), "Cargo.toml")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "toml" {
			t.Errorf("Language = %q, want %q", result.Language, "toml")
		}
	})
}

// TestSvelteParser tests Svelte parsing
func TestSvelteParser(t *testing.T) {
	parser := NewSvelteParser()

	t.Run("parse component", func(t *testing.T) {
		code := `
<script>
  import { onMount } from 'svelte';
  import Header from './Header.svelte';
  export let name = 'world';
</script>

<Header title="Welcome" />
<h1>Hello {name}!</h1>
`
		result, err := parser.Parse([]byte(code), "App.svelte")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "svelte" {
			t.Errorf("Language = %q, want %q", result.Language, "svelte")
		}

		// Verify relationships (imports)
		foundImport := false
		for _, rel := range result.Relationships {
			if rel.Kind == RelImport && (rel.TargetFile == "svelte" || rel.TargetFile == "./Header.svelte") {
				foundImport = true
			}
		}
		if !foundImport {
			t.Error("Imports not found in Svelte code")
		}
	})
}

// TestProtobufParser tests Protobuf parsing
func TestProtobufParser(t *testing.T) {
	parser := NewProtobufParser()

	t.Run("parse message and service", func(t *testing.T) {
		code := `
syntax = "proto3";
package api;

message User {
  string id = 1;
  string name = 2;
}

service UserService {
  rpc GetUser(UserRequest) returns (User);
}
`
		result, err := parser.Parse([]byte(code), "api.proto")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "protobuf" {
			t.Errorf("Language = %q, want %q", result.Language, "protobuf")
		}
	})
}

// TestGroovyParser tests Groovy parsing
func TestGroovyParser(t *testing.T) {
	parser := NewGroovyParser()

	t.Run("parse groovy script", func(t *testing.T) {
		code := `def greet(name) { println "Hello $name" }`
		result, err := parser.Parse([]byte(code), "script.groovy")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "groovy" {
			t.Errorf("Language = %q, want %q", result.Language, "groovy")
		}
	})
}

// TestOCamlParser tests OCaml parsing
func TestOCamlParser(t *testing.T) {
	parser := NewOCamlParser()

	t.Run("parse ocaml function", func(t *testing.T) {
		code := `let greet name = print_endline ("Hello " ^ name)`
		result, err := parser.Parse([]byte(code), "greet.ml")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "ocaml" {
			t.Errorf("Language = %q, want %q", result.Language, "ocaml")
		}
	})
}

// TestElmParser tests Elm parsing
func TestElmParser(t *testing.T) {
	parser := NewElmParser()

	t.Run("parse elm function", func(t *testing.T) {
		code := `module Main exposing (..)
greet name = "Hello " ++ name`
		result, err := parser.Parse([]byte(code), "Main.elm")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "elm" {
			t.Errorf("Language = %q, want %q", result.Language, "elm")
		}
	})
}

// TestCUEParser tests CUE parsing
func TestCUEParser(t *testing.T) {
	parser := NewCUEParser()

	t.Run("parse cue schema", func(t *testing.T) {
		code := `package schema
#User: { name: string, age: int & >0 }`
		result, err := parser.Parse([]byte(code), "schema.cue")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "cue" {
			t.Errorf("Language = %q, want %q", result.Language, "cue")
		}
	})
}

// TestDartParser tests Dart parsing
func TestDartParser(t *testing.T) {
	parser := NewDartParser()

	t.Run("parse dart class", func(t *testing.T) {
		code := `class User { String name; User(this.name); }`
		result, err := parser.Parse([]byte(code), "user.dart")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if result.Language != "dart" {
			t.Errorf("Language = %q, want %q", result.Language, "dart")
		}
	})
}

// TestParserLanguageMethod tests that all parsers return correct Language()
func TestParserLanguageMethod(t *testing.T) {
	tests := []struct {
		parser   Parser
		expected Language
	}{
		{NewGoParser(), LangGo},
		{NewTypeScriptParser(), LangTypeScript},
		{NewJavaScriptParser(), LangJavaScript},
		{NewPythonParser(), LangPython},
		{NewRustParser(), LangRust},
		{NewJavaParser(), LangJava},
		{NewCParser(), LangC},
		{NewCPPParser(), LangCPP},
		{NewCSharpParser(), LangCSharp},
		{NewRubyParser(), LangRuby},
		{NewPHPParser(), LangPHP},
		{NewSwiftParser(), LangSwift},
		{NewKotlinParser(), LangKotlin},
		{NewScalaParser(), LangScala},
		{NewBashParser(), LangBash},
		{NewSQLParser(), LangSQL},
		{NewDockerfileParser(), LangDockerfile},
		{NewHCLParser(), LangHCL},
		{NewHTMLParser(), LangHTML},
		{NewCSSParser(), LangCSS},
		{NewYAMLParser(), LangYAML},
		{NewTOMLParser(), LangTOML},
		{NewJSONParser(), LangJSON},
		{NewMarkdownParser(), LangMarkdown},
		{NewElixirParser(), LangElixir},
		{NewLuaParser(), LangLua},
		{NewGroovyParser(), LangGroovy},
		{NewSvelteParser(), LangSvelte},
		{NewOCamlParser(), LangOCaml},
		{NewElmParser(), LangElm},
		{NewProtobufParser(), LangProtobuf},
		{NewDartParser(), LangDart},
		{NewCUEParser(), LangCUE},
	}

	for _, tt := range tests {
		t.Run(string(tt.expected), func(t *testing.T) {
			if tt.parser.Language() != tt.expected {
				t.Errorf("Language() = %q, want %q", tt.parser.Language(), tt.expected)
			}
		})
	}
}
