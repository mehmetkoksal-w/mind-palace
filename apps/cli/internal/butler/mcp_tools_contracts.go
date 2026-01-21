package butler

import (
	"fmt"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

// toolContractsGet retrieves contracts from the store with filtering.
func (s *MCPServer) toolContractsGet(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	store := contracts.NewStore(mem.DB())

	// Ensure tables exist (defensive for fresh DB)
	if err := store.CreateTables(); err != nil {
		return s.toolError(id, fmt.Sprintf("initialize contracts failed: %v", err))
	}

	// Parse filters
	filter := contracts.ContractFilter{
		Limit: 50,
	}

	if method, ok := args["method"].(string); ok && method != "" {
		filter.Method = method
	}
	if status, ok := args["status"].(string); ok && status != "" {
		filter.Status = contracts.ContractStatus(status)
	}
	if endpoint, ok := args["endpoint"].(string); ok && endpoint != "" {
		filter.Endpoint = endpoint
	}
	if hasMismatches, ok := args["has_mismatches"].(bool); ok && hasMismatches {
		filter.HasMismatches = true
	}
	if limit, ok := args["limit"].(float64); ok && limit > 0 {
		filter.Limit = int(limit)
	}

	contractList, err := store.ListContracts(filter)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get contracts failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# API Contracts\n\n")

	if len(contractList) == 0 {
		output.WriteString("No contracts found matching the criteria.\n\n")
		output.WriteString("Run `palace contracts scan` to detect API contracts in the codebase.\n")
	} else {
		// Group by status
		discovered := 0
		verified := 0
		mismatch := 0
		ignored := 0
		for _, c := range contractList {
			switch c.Status {
			case contracts.ContractDiscovered:
				discovered++
			case contracts.ContractVerified:
				verified++
			case contracts.ContractMismatch:
				mismatch++
			case contracts.ContractIgnored:
				ignored++
			}
		}

		fmt.Fprintf(&output, "**Found:** %d contracts (%d discovered, %d verified, %d mismatch, %d ignored)\n\n",
			len(contractList), discovered, verified, mismatch, ignored)

		// Group by method
		byMethod := make(map[string][]*contracts.Contract)
		for _, c := range contractList {
			byMethod[c.Method] = append(byMethod[c.Method], c)
		}

		methodOrder := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
		for _, method := range methodOrder {
			ctrs := byMethod[method]
			if len(ctrs) == 0 {
				continue
			}
			fmt.Fprintf(&output, "## %s\n\n", method)
			for _, c := range ctrs {
				statusIcon := "🔵"
				switch c.Status {
				case contracts.ContractVerified:
					statusIcon = "✅"
				case contracts.ContractMismatch:
					statusIcon = "⚠️"
				case contracts.ContractIgnored:
					statusIcon = "⬜"
				}

				mismatchInfo := ""
				if len(c.Mismatches) > 0 {
					errors := c.ErrorCount()
					warnings := c.WarningCount()
					if errors > 0 {
						mismatchInfo = fmt.Sprintf(" [%d errors]", errors)
					}
					if warnings > 0 {
						mismatchInfo += fmt.Sprintf(" [%d warnings]", warnings)
					}
				}

				fmt.Fprintf(&output, "%s `%s` (`%s`)%s\n", statusIcon, c.Endpoint, c.ID, mismatchInfo)
				fmt.Fprintf(&output, "   Calls: %d | Confidence: %.0f%% | Status: %s\n\n",
					len(c.FrontendCalls), c.Confidence*100, c.Status)
			}
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

// toolContractShow shows details of a specific contract.
func (s *MCPServer) toolContractShow(id any, args map[string]interface{}) jsonRPCResponse {
	contractID, _ := args["contract_id"].(string)
	if contractID == "" {
		return s.toolError(id, "contract_id is required")
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	store := contracts.NewStore(mem.DB())

	// Ensure tables exist (defensive for fresh DB)
	if err := store.CreateTables(); err != nil {
		return s.toolError(id, fmt.Sprintf("initialize contracts failed: %v", err))
	}

	contract, err := store.GetContract(contractID)
	if err != nil || contract == nil {
		return s.toolError(id, fmt.Sprintf("contract not found: %s", contractID))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Contract: %s %s\n\n", contract.Method, contract.Endpoint)

	statusIcon := "🔵"
	switch contract.Status {
	case contracts.ContractVerified:
		statusIcon = "✅"
	case contracts.ContractMismatch:
		statusIcon = "⚠️"
	case contracts.ContractIgnored:
		statusIcon = "⬜"
	}

	fmt.Fprintf(&output, "**ID:** `%s`\n", contract.ID)
	fmt.Fprintf(&output, "**Status:** %s %s\n", statusIcon, contract.Status)
	fmt.Fprintf(&output, "**Pattern:** `%s`\n", contract.EndpointPattern)
	fmt.Fprintf(&output, "**Confidence:** %.1f%%\n\n", contract.Confidence*100)

	// Backend info
	output.WriteString("## Backend\n\n")
	fmt.Fprintf(&output, "| Field | Value |\n")
	fmt.Fprintf(&output, "|-------|-------|\n")
	fmt.Fprintf(&output, "| File | `%s:%d` |\n", contract.Backend.File, contract.Backend.Line)
	fmt.Fprintf(&output, "| Framework | %s |\n", contract.Backend.Framework)
	fmt.Fprintf(&output, "| Handler | %s |\n", contract.Backend.Handler)
	output.WriteString("\n")

	if contract.Backend.ResponseSchema != nil {
		fmt.Fprintf(&output, "**Response Schema:** %s\n\n", contract.Backend.ResponseSchema.String())
	}

	// Frontend calls
	if len(contract.FrontendCalls) > 0 {
		fmt.Fprintf(&output, "## Frontend Calls (%d)\n\n", len(contract.FrontendCalls))
		shown := 0
		for _, call := range contract.FrontendCalls {
			if shown >= 5 {
				fmt.Fprintf(&output, "... and %d more\n", len(contract.FrontendCalls)-shown)
				break
			}
			fmt.Fprintf(&output, "- `%s:%d` (%s)\n", call.File, call.Line, call.CallType)
			shown++
		}
		output.WriteString("\n")
	}

	// Mismatches
	if len(contract.Mismatches) > 0 {
		fmt.Fprintf(&output, "## Mismatches (%d errors, %d warnings)\n\n",
			contract.ErrorCount(), contract.WarningCount())

		for _, m := range contract.Mismatches {
			severityIcon := "⚠️"
			if m.Severity == contracts.SeverityError {
				severityIcon = "❌"
			}
			fmt.Fprintf(&output, "%s **%s:** %s\n", severityIcon, m.FieldPath, m.Description)
			if m.BackendType != "" && m.FrontendType != "" {
				fmt.Fprintf(&output, "   Backend: `%s`, Frontend: `%s`\n", m.BackendType, m.FrontendType)
			}
			output.WriteString("\n")
		}
	}

	fmt.Fprintf(&output, "---\n")
	fmt.Fprintf(&output, "**First seen:** %s\n", contract.FirstSeen.Format("2006-01-02 15:04"))
	fmt.Fprintf(&output, "**Last seen:** %s\n", contract.LastSeen.Format("2006-01-02 15:04"))

	if contract.Status == contracts.ContractDiscovered || contract.Status == contracts.ContractMismatch {
		output.WriteString("\n---\n")
		output.WriteString("Use `contract_verify` to mark this contract as verified or `contract_ignore` to ignore it.\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolContractVerify marks a contract as verified.
func (s *MCPServer) toolContractVerify(id any, args map[string]interface{}) jsonRPCResponse {
	contractID, _ := args["contract_id"].(string)
	if contractID == "" {
		return s.toolError(id, "contract_id is required")
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	store := contracts.NewStore(mem.DB())

	// Ensure tables exist (defensive for fresh DB)
	if err := store.CreateTables(); err != nil {
		return s.toolError(id, fmt.Sprintf("initialize contracts failed: %v", err))
	}

	contract, err := store.GetContract(contractID)
	if err != nil || contract == nil {
		return s.toolError(id, fmt.Sprintf("contract not found: %s", contractID))
	}

	if contract.Status == contracts.ContractVerified {
		return s.toolError(id, fmt.Sprintf("contract %s is already verified", contractID))
	}

	// Clear mismatches if any
	if len(contract.Mismatches) > 0 {
		if err := store.ClearMismatches(contractID); err != nil {
			return s.toolError(id, fmt.Sprintf("clear mismatches failed: %v", err))
		}
	}

	if err := store.UpdateStatus(contractID, contracts.ContractVerified); err != nil {
		return s.toolError(id, fmt.Sprintf("verify failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Contract Verified\n\n")
	fmt.Fprintf(&output, "**Contract:** `%s`\n", contractID)
	fmt.Fprintf(&output, "**Endpoint:** %s %s\n\n", contract.Method, contract.Endpoint)
	output.WriteString("The contract has been marked as verified.\n")
	if len(contract.Mismatches) > 0 {
		fmt.Fprintf(&output, "%d mismatches were cleared.\n", len(contract.Mismatches))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolContractIgnore marks a contract as ignored.
func (s *MCPServer) toolContractIgnore(id any, args map[string]interface{}) jsonRPCResponse {
	contractID, _ := args["contract_id"].(string)
	if contractID == "" {
		return s.toolError(id, "contract_id is required")
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	store := contracts.NewStore(mem.DB())

	// Ensure tables exist (defensive for fresh DB)
	if err := store.CreateTables(); err != nil {
		return s.toolError(id, fmt.Sprintf("initialize contracts failed: %v", err))
	}

	contract, err := store.GetContract(contractID)
	if err != nil || contract == nil {
		return s.toolError(id, fmt.Sprintf("contract not found: %s", contractID))
	}

	if contract.Status == contracts.ContractIgnored {
		return s.toolError(id, fmt.Sprintf("contract %s is already ignored", contractID))
	}

	if err := store.UpdateStatus(contractID, contracts.ContractIgnored); err != nil {
		return s.toolError(id, fmt.Sprintf("ignore failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Contract Ignored\n\n")
	fmt.Fprintf(&output, "**Contract:** `%s`\n", contractID)
	fmt.Fprintf(&output, "**Endpoint:** %s %s\n\n", contract.Method, contract.Endpoint)
	output.WriteString("The contract has been marked as ignored and will not appear in future results.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolContractStats returns statistics about API contracts.
func (s *MCPServer) toolContractStats(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	store := contracts.NewStore(mem.DB())

	// Ensure tables exist (defensive for fresh DB)
	if err := store.CreateTables(); err != nil {
		return s.toolError(id, fmt.Sprintf("initialize contracts failed: %v", err))
	}

	stats, err := store.GetStats()
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get stats failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Contract Statistics\n\n")

	fmt.Fprintf(&output, "| Metric | Value |\n")
	fmt.Fprintf(&output, "|--------|-------|\n")
	fmt.Fprintf(&output, "| Total Contracts | %d |\n", stats.Total)
	fmt.Fprintf(&output, "| Discovered | %d |\n", stats.Discovered)
	fmt.Fprintf(&output, "| Verified | %d |\n", stats.Verified)
	fmt.Fprintf(&output, "| Mismatch | %d |\n", stats.Mismatch)
	fmt.Fprintf(&output, "| Ignored | %d |\n", stats.Ignored)
	fmt.Fprintf(&output, "| Total Errors | %d |\n", stats.TotalErrors)
	fmt.Fprintf(&output, "| Total Warnings | %d |\n", stats.TotalWarnings)
	fmt.Fprintf(&output, "| Total Frontend Calls | %d |\n", stats.TotalCalls)
	output.WriteString("\n")

	if len(stats.ByMethod) > 0 {
		output.WriteString("## By HTTP Method\n\n")
		fmt.Fprintf(&output, "| Method | Count |\n")
		fmt.Fprintf(&output, "|--------|-------|\n")
		methodOrder := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
		for _, method := range methodOrder {
			if count, ok := stats.ByMethod[method]; ok && count > 0 {
				fmt.Fprintf(&output, "| %s | %d |\n", method, count)
			}
		}
		output.WriteString("\n")
	}

	if stats.Mismatch > 0 {
		output.WriteString("---\n")
		fmt.Fprintf(&output, "**%d contracts** have mismatches that need attention.\n", stats.Mismatch)
		output.WriteString("Use `contracts_get` with `has_mismatches: true` to see them.\n")
	}

	if stats.Discovered > 0 {
		fmt.Fprintf(&output, "**%d contracts** are waiting for review.\n", stats.Discovered)
		output.WriteString("Use `contracts_get` with `status: discovered` to see them.\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolContractMismatches returns all contracts with type mismatches.
func (s *MCPServer) toolContractMismatches(id any, args map[string]interface{}) jsonRPCResponse {
	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	store := contracts.NewStore(mem.DB())

	// Ensure tables exist (defensive for fresh DB)
	if err := store.CreateTables(); err != nil {
		return s.toolError(id, fmt.Sprintf("initialize contracts failed: %v", err))
	}

	contractList, err := store.ListContracts(contracts.ContractFilter{
		HasMismatches: true,
		Limit:         100,
	})
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get contracts failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# API Type Mismatches\n\n")

	if len(contractList) == 0 {
		output.WriteString("No type mismatches found.\n\n")
		output.WriteString("All FE-BE contracts have matching types. Great work!\n")
	} else {
		totalErrors := 0
		totalWarnings := 0
		for _, c := range contractList {
			totalErrors += c.ErrorCount()
			totalWarnings += c.WarningCount()
		}

		fmt.Fprintf(&output, "**Found:** %d contracts with mismatches (%d errors, %d warnings)\n\n",
			len(contractList), totalErrors, totalWarnings)

		for _, c := range contractList {
			fmt.Fprintf(&output, "## %s %s\n\n", c.Method, c.Endpoint)
			fmt.Fprintf(&output, "**ID:** `%s` | **Calls:** %d\n\n", c.ID, len(c.FrontendCalls))

			for _, m := range c.Mismatches {
				severityIcon := "⚠️"
				if m.Severity == contracts.SeverityError {
					severityIcon = "❌"
				}
				fmt.Fprintf(&output, "- %s **%s:** %s\n", severityIcon, m.FieldPath, m.Description)
				if m.BackendType != "" && m.FrontendType != "" {
					fmt.Fprintf(&output, "  - Backend: `%s`, Frontend: `%s`\n", m.BackendType, m.FrontendType)
				}
			}
			output.WriteString("\n")
		}

		output.WriteString("---\n")
		output.WriteString("Use `contract_verify` to mark contracts as verified after fixing mismatches.\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}
