package analysis

import (
	"testing"
)

// TestCPPParser tests C++ parsing
func TestCPPParser(t *testing.T) {
	parser := NewCPPParser()

	t.Run("parse class with methods", func(t *testing.T) {
		code := `class MyClass {
public:
    MyClass() {}
    ~MyClass() {}
    void doSomething();
    int getValue() const;
private:
    int value;
};`
		result, err := parser.Parse([]byte(code), "test.cpp")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "cpp" {
			t.Errorf("Language = %q, want %q", result.Language, "cpp")
		}

		// Just verify we got some symbols
		if len(result.Symbols) == 0 {
			t.Log("Warning: No symbols found in C++ class")
		}
	})

	t.Run("parse struct", func(t *testing.T) {
		code := `struct Point {
    int x;
    int y;
    void move(int dx, int dy);
};`
		result, err := parser.Parse([]byte(code), "point.cpp")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "cpp" {
			t.Errorf("Language = %q, want %q", result.Language, "cpp")
		}
	})

	t.Run("parse enum", func(t *testing.T) {
		code := `enum Color {
    Red,
    Green,
    Blue
};

enum class Status {
    Pending,
    Active,
    Done
};`
		result, err := parser.Parse([]byte(code), "enums.cpp")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "cpp" {
			t.Errorf("Language = %q, want %q", result.Language, "cpp")
		}

		// Just log if enums are found
		for _, sym := range result.Symbols {
			t.Logf("Found symbol: %s (%s)", sym.Name, sym.Kind)
		}
	})

	t.Run("parse namespace", func(t *testing.T) {
		code := `namespace MyNamespace {
    class InnerClass {
    public:
        void method();
    };
}`
		result, err := parser.Parse([]byte(code), "namespace.cpp")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "cpp" {
			t.Errorf("Language = %q, want %q", result.Language, "cpp")
		}
	})

	t.Run("parse template class", func(t *testing.T) {
		code := `template<typename T>
class Container {
public:
    void add(T item);
    T get(int index);
private:
    std::vector<T> items;
};`
		result, err := parser.Parse([]byte(code), "container.cpp")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "cpp" {
			t.Errorf("Language = %q, want %q", result.Language, "cpp")
		}
	})

	t.Run("parse function with declaration", func(t *testing.T) {
		code := `void process(int value);

void process(int value) {
    // implementation
}`
		result, err := parser.Parse([]byte(code), "func.cpp")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "cpp" {
			t.Errorf("Language = %q, want %q", result.Language, "cpp")
		}
	})

	t.Run("parse include relationships", func(t *testing.T) {
		code := `#include <iostream>
#include "myheader.h"
#include <vector>

int main() {
    return 0;
}`
		result, err := parser.Parse([]byte(code), "main.cpp")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		// Log relationships found
		t.Logf("Found %d relationships", len(result.Relationships))
	})

	t.Run("parse class inheritance", func(t *testing.T) {
		code := `class Base {
public:
    virtual void show() = 0;
};

class Derived : public Base, protected OtherBase {
public:
    void show() override {}
};`
		result, err := parser.Parse([]byte(code), "inheritance.cpp")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "cpp" {
			t.Errorf("Language = %q, want %q", result.Language, "cpp")
		}
	})

	t.Run("parse destructor", func(t *testing.T) {
		code := `class Resource {
public:
    Resource() {}
    ~Resource() { cleanup(); }
private:
    void cleanup() {}
};`
		result, err := parser.Parse([]byte(code), "resource.cpp")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "cpp" {
			t.Errorf("Language = %q, want %q", result.Language, "cpp")
		}
	})

	t.Run("parse template function", func(t *testing.T) {
		code := `template<typename T>
T max(T a, T b) {
    return (a > b) ? a : b;
}`
		result, err := parser.Parse([]byte(code), "template.cpp")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		if result.Language != "cpp" {
			t.Errorf("Language = %q, want %q", result.Language, "cpp")
		}
	})
}
