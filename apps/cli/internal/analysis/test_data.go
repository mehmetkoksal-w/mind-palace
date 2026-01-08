package analysis

// Test constants and helper data shared between unit and integration tests

// Sample Go code for testing parsers
const testGoCode = `package main

import (
	"fmt"
	"strings"
)

// Person represents a person
type Person struct {
	Name string
	Age  int
}

// NewPerson creates a new person
func NewPerson(name string, age int) *Person {
	return &Person{
		Name: name,
		Age:  age,
	}
}

// Greet says hello
func (p *Person) Greet() string {
	return fmt.Sprintf("Hello, I'm %s", p.Name)
}

// IsAdult checks if person is adult
func (p *Person) IsAdult() bool {
	return p.Age >= 18
}

const MaxAge = 120

var defaultPerson = Person{
	Name: "Unknown",
	Age:  0,
}

func main() {
	person := NewPerson("Alice", 25)
	fmt.Println(person.Greet())
	if person.IsAdult() {
		fmt.Println("Adult")
	}
}
`
