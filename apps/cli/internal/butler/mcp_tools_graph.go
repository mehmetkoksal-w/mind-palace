package butler

import (
	"fmt"
	"strings"
)

// toolExploreCallers finds all locations that call a function or method.
func (s *MCPServer) toolExploreCallers(id any, args map[string]interface{}) jsonRPCResponse {
	symbol, _ := args["symbol"].(string)
	if symbol == "" {
		return s.toolError(id, "symbol is required")
	}

	calls, err := s.butler.GetIncomingCalls(symbol)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get callers failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Callers of `%s`\n\n", symbol)

	if len(calls) == 0 {
		output.WriteString("No callers found. This symbol may not be called anywhere, or call tracking may not be available for this language.\n")
	} else {
		fmt.Fprintf(&output, "Found %d call sites:\n\n", len(calls))
		for _, call := range calls {
			fmt.Fprintf(&output, "- `%s` line %d", call.FilePath, call.Line)
			if call.CallerSymbol != "" {
				fmt.Fprintf(&output, " (in function `%s`)", call.CallerSymbol)
			}
			output.WriteString("\n")
		}
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolExploreCallees finds all functions/methods called by a symbol.
func (s *MCPServer) toolExploreCallees(id any, args map[string]interface{}) jsonRPCResponse {
	symbol, _ := args["symbol"].(string)
	if symbol == "" {
		return s.toolError(id, "symbol is required")
	}

	file, _ := args["file"].(string)
	if file == "" {
		return s.toolError(id, "file is required to find the symbol's scope")
	}

	calls, err := s.butler.GetOutgoingCalls(symbol, file)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get callees failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Functions called by `%s`\n\n", symbol)
	fmt.Fprintf(&output, "File: `%s`\n\n", file)

	if len(calls) == 0 {
		output.WriteString("No outgoing calls found. This function may not call other functions, or call tracking may not be available.\n")
	} else {
		fmt.Fprintf(&output, "Found %d function calls:\n\n", len(calls))
		for _, call := range calls {
			fmt.Fprintf(&output, "- `%s` (line %d)\n", call.CalleeSymbol, call.Line)
		}
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolExploreGraph gets the complete call graph for a file.
func (s *MCPServer) toolExploreGraph(id any, args map[string]interface{}) jsonRPCResponse {
	file, _ := args["file"].(string)
	if file == "" {
		return s.toolError(id, "file is required")
	}

	graph, err := s.butler.GetCallGraph(file)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get call graph failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Call Graph for `%s`\n\n", file)

	output.WriteString("## Incoming Calls (who calls functions in this file)\n\n")
	if len(graph.IncomingCalls) == 0 {
		output.WriteString("No incoming calls from other files.\n\n")
	} else {
		for _, call := range graph.IncomingCalls {
			fmt.Fprintf(&output, "- `%s` called from `%s` line %d", call.CalleeSymbol, call.FilePath, call.Line)
			if call.CallerSymbol != "" {
				fmt.Fprintf(&output, " (in `%s`)", call.CallerSymbol)
			}
			output.WriteString("\n")
		}
		output.WriteString("\n")
	}

	output.WriteString("## Outgoing Calls (what this file calls)\n\n")
	if len(graph.OutgoingCalls) == 0 {
		output.WriteString("No outgoing calls tracked.\n\n")
	} else {
		for _, call := range graph.OutgoingCalls {
			fmt.Fprintf(&output, "- `%s` at line %d", call.CalleeSymbol, call.Line)
			if call.CallerSymbol != "" {
				fmt.Fprintf(&output, " (from `%s`)", call.CallerSymbol)
			}
			output.WriteString("\n")
		}
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}
