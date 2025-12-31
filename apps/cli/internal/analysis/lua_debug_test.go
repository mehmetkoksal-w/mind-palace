package analysis

import (
	"context"
	"fmt"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/lua"
)

func TestLuaAST(t *testing.T) {
	code := `
function greet(name)
  print("Hello, " .. name)
end

local function secret()
  return 42
end

local x = 10
y = 20
config.port = 8080
`
	parser := sitter.NewParser()
	parser.SetLanguage(lua.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte(code))

	printNodes(tree.RootNode(), 0)
}

func printNodes(n *sitter.Node, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	fmt.Printf("%s%s\n", indent, n.Type())
	for i := 0; i < int(n.ChildCount()); i++ {
		printNodes(n.Child(i), depth+1)
	}
}
