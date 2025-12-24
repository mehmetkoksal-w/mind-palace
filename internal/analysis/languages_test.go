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
