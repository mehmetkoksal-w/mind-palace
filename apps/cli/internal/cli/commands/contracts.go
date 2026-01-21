package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli/flags"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts/extractors"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "contracts",
		Aliases:     []string{"contract", "ctr"},
		Description: "Manage FE-BE API contracts",
		Run:         RunContracts,
	})
}

// ContractsOptions contains the configuration for the contracts command.
type ContractsOptions struct {
	Root         string
	Subcommand   string
	Method       string
	Status       string
	Endpoint     string
	HasMismatches bool
	Limit        int
	ContractID   string
	BackendDir   string
	FrontendDir  string
}

// RunContracts executes the contracts command.
func RunContracts(args []string) error {
	if len(args) == 0 {
		return showContractsHelp()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "scan":
		return RunContractsScan(subArgs)
	case "list", "ls":
		return RunContractsList(subArgs)
	case "show":
		return RunContractsShow(subArgs)
	case "verify":
		return RunContractsVerify(subArgs)
	case "ignore":
		return RunContractsIgnore(subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\nRun 'palace contracts' for usage", subcommand)
	}
}

func showContractsHelp() error {
	fmt.Println(`Usage: palace contracts <subcommand> [options]

Manage FE-BE API contracts.

Subcommands:
  scan      Scan codebase for API contracts
  list      List detected contracts
  show      Show contract details
  verify    Mark a contract as verified
  ignore    Ignore a contract

Examples:
  palace contracts scan
  palace contracts scan --backend ./api --frontend ./web
  palace contracts list --status mismatch
  palace contracts list --method GET
  palace contracts show ctr_abc123
  palace contracts verify ctr_abc123
  palace contracts ignore ctr_xyz789`)
	return nil
}

// RunContractsScan scans for API contracts.
func RunContractsScan(args []string) error {
	fs := flag.NewFlagSet("contracts scan", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	backendDir := fs.String("backend", "", "backend source directory (defaults to project root)")
	frontendDir := fs.String("frontend", "", "frontend source directory (defaults to project root)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecuteContractsScan(ContractsOptions{
		Root:        *root,
		BackendDir:  *backendDir,
		FrontendDir: *frontendDir,
	})
}

// ExecuteContractsScan scans the codebase for API contracts.
func ExecuteContractsScan(opts ContractsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	// Verify palace is initialized
	palacePath := filepath.Join(rootPath, ".palace")
	if _, err := os.Stat(palacePath); os.IsNotExist(err) {
		return fmt.Errorf("palace not initialized in %s. Run 'palace init' first", rootPath)
	}

	fmt.Println("Scanning for API contracts...")

	// Open memory
	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	// Initialize contract store
	store := contracts.NewStore(mem.DB())
	if err := store.CreateTables(); err != nil {
		return fmt.Errorf("create contract tables: %w", err)
	}

	// Determine directories to scan
	backendDir := rootPath
	if opts.BackendDir != "" {
		backendDir = filepath.Join(rootPath, opts.BackendDir)
	}
	frontendDir := rootPath
	if opts.FrontendDir != "" {
		frontendDir = filepath.Join(rootPath, opts.FrontendDir)
	}

	// Collect backend files
	backendFiles, err := collectBackendFiles(backendDir)
	if err != nil {
		return fmt.Errorf("collect backend files: %w", err)
	}
	fmt.Printf("Found %d backend files to analyze\n", len(backendFiles))

	// Collect frontend files
	frontendFiles, err := collectFrontendFiles(frontendDir)
	if err != nil {
		return fmt.Errorf("collect frontend files: %w", err)
	}
	fmt.Printf("Found %d frontend files to analyze\n", len(frontendFiles))

	// Extract endpoints from backend
	endpoints, err := extractEndpoints(backendFiles)
	if err != nil {
		return fmt.Errorf("extract endpoints: %w", err)
	}
	fmt.Printf("Found %d backend endpoints\n", len(endpoints))

	// Extract API calls from frontend
	calls, err := extractCalls(frontendFiles)
	if err != nil {
		return fmt.Errorf("extract calls: %w", err)
	}
	fmt.Printf("Found %d frontend API calls\n", len(calls))

	// Analyze contracts
	analyzer := contracts.NewAnalyzer()
	input := &contracts.AnalysisInput{
		Endpoints: endpoints,
		Calls:     calls,
	}
	result := analyzer.Analyze(input)

	// Save contracts
	for _, contract := range result.Contracts {
		if err := store.SaveContract(contract); err != nil {
			return fmt.Errorf("save contract: %w", err)
		}
	}

	// Print summary
	fmt.Println()
	fmt.Println("Scan complete")
	fmt.Printf("  Contracts discovered: %d\n", len(result.Contracts))
	fmt.Printf("  Unmatched backend endpoints: %d\n", len(result.UnmatchedBackend))
	fmt.Printf("  Unmatched frontend calls: %d\n", len(result.UnmatchedFrontend))

	// Count mismatches
	mismatchCount := 0
	for _, c := range result.Contracts {
		if len(c.Mismatches) > 0 {
			mismatchCount++
		}
	}
	if mismatchCount > 0 {
		fmt.Printf("  Contracts with mismatches: %d\n", mismatchCount)
	}

	// Show discovered contracts
	if len(result.Contracts) > 0 {
		fmt.Println()
		fmt.Println("Discovered contracts:")
		for _, c := range result.Contracts {
			icon := "+"
			if len(c.Mismatches) > 0 {
				icon = "!"
			}
			fmt.Printf("  [%s] %s %s (%.0f%%) - %d frontend calls\n",
				icon, c.Method, c.Endpoint, c.Confidence*100, len(c.FrontendCalls))
		}
		fmt.Println()
		fmt.Println("Use 'palace contracts list' to see all contracts")
		fmt.Println("Use 'palace contracts show <id>' to see contract details")
	}

	// Show unmatched backend endpoints
	if len(result.UnmatchedBackend) > 0 {
		fmt.Println()
		fmt.Println("Unmatched backend endpoints (no frontend calls found):")
		shown := 0
		for _, ep := range result.UnmatchedBackend {
			if shown >= 5 {
				fmt.Printf("  ... and %d more\n", len(result.UnmatchedBackend)-shown)
				break
			}
			fmt.Printf("  - %s %s\n", ep.Method, ep.Path)
			shown++
		}
	}

	// Show unmatched frontend calls
	if len(result.UnmatchedFrontend) > 0 {
		fmt.Println()
		fmt.Println("Unmatched frontend calls (no backend endpoint found):")
		shown := 0
		for _, call := range result.UnmatchedFrontend {
			if shown >= 5 {
				fmt.Printf("  ... and %d more\n", len(result.UnmatchedFrontend)-shown)
				break
			}
			fmt.Printf("  - %s %s (%s:%d)\n", call.Method, call.URL, call.File, call.Line)
			shown++
		}
	}

	return nil
}

// RunContractsList lists detected contracts.
func RunContractsList(args []string) error {
	fs := flag.NewFlagSet("contracts list", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	method := fs.String("method", "", "filter by HTTP method (GET, POST, PUT, DELETE, PATCH)")
	status := fs.String("status", "", "filter by status: discovered, verified, mismatch, ignored")
	endpoint := fs.String("endpoint", "", "filter by endpoint pattern")
	hasMismatches := fs.Bool("mismatches", false, "only show contracts with mismatches")
	limit := fs.Int("limit", 50, "maximum contracts to show")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecuteContractsList(ContractsOptions{
		Root:          *root,
		Method:        *method,
		Status:        *status,
		Endpoint:      *endpoint,
		HasMismatches: *hasMismatches,
		Limit:         *limit,
	})
}

// ExecuteContractsList lists contracts matching criteria.
func ExecuteContractsList(opts ContractsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	store := contracts.NewStore(mem.DB())

	contractList, err := store.ListContracts(contracts.ContractFilter{
		Method:        opts.Method,
		Status:        contracts.ContractStatus(opts.Status),
		Endpoint:      opts.Endpoint,
		HasMismatches: opts.HasMismatches,
		Limit:         opts.Limit,
	})
	if err != nil {
		return fmt.Errorf("list contracts: %w", err)
	}

	if len(contractList) == 0 {
		fmt.Println("No contracts found.")
		fmt.Println("Run 'palace contracts scan' to detect contracts.")
		return nil
	}

	// Get stats
	stats, err := store.GetStats()
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	fmt.Printf("Contracts: %d total (%d discovered, %d verified, %d mismatch, %d ignored)\n",
		stats.Total, stats.Discovered, stats.Verified, stats.Mismatch, stats.Ignored)
	if stats.TotalErrors > 0 || stats.TotalWarnings > 0 {
		fmt.Printf("Issues: %d errors, %d warnings\n", stats.TotalErrors, stats.TotalWarnings)
	}
	fmt.Println(strings.Repeat("=", 70))

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
		fmt.Printf("\n%s:\n", method)
		for _, c := range ctrs {
			statusIcon := "?"
			switch c.Status {
			case contracts.ContractDiscovered:
				statusIcon = "?"
			case contracts.ContractVerified:
				statusIcon = "+"
			case contracts.ContractMismatch:
				statusIcon = "!"
			case contracts.ContractIgnored:
				statusIcon = "x"
			}

			mismatchInfo := ""
			if len(c.Mismatches) > 0 {
				errCount := c.ErrorCount()
				warnCount := c.WarningCount()
				if errCount > 0 {
					mismatchInfo = fmt.Sprintf(" [%d errors]", errCount)
				}
				if warnCount > 0 {
					mismatchInfo += fmt.Sprintf(" [%d warnings]", warnCount)
				}
			}

			fmt.Printf("  [%s] %s  %s%s\n", statusIcon, c.ID, c.Endpoint, mismatchInfo)
			fmt.Printf("      Calls: %d | Confidence: %.0f%% | Status: %s\n",
				len(c.FrontendCalls), c.Confidence*100, c.Status)
		}
	}

	fmt.Println()
	fmt.Println("Use 'palace contracts show <id>' to see contract details")
	fmt.Println("Use 'palace contracts verify <id>' to mark as verified")

	return nil
}

// RunContractsShow shows details of a contract.
func RunContractsShow(args []string) error {
	fs := flag.NewFlagSet("contracts show", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New(`usage: palace contracts show <contract-id>

Show detailed information about a contract.

Arguments:
  <contract-id>    ID of contract to show (e.g., ctr_abc123)`)
	}

	return ExecuteContractsShow(ContractsOptions{
		Root:       *root,
		ContractID: remaining[0],
	})
}

// ExecuteContractsShow shows contract details.
func ExecuteContractsShow(opts ContractsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	store := contracts.NewStore(mem.DB())

	contract, err := store.GetContract(opts.ContractID)
	if err != nil {
		return fmt.Errorf("get contract: %w", err)
	}
	if contract == nil {
		return fmt.Errorf("contract not found: %s", opts.ContractID)
	}

	// Display
	fmt.Printf("Contract: %s\n", contract.ID)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Endpoint:    %s %s\n", contract.Method, contract.Endpoint)
	fmt.Printf("Pattern:     %s\n", contract.EndpointPattern)
	fmt.Println()
	fmt.Printf("Status:      %s\n", contract.Status)
	fmt.Printf("Authority:   %s\n", contract.Authority)
	fmt.Printf("Confidence:  %.1f%%\n", contract.Confidence*100)
	fmt.Println()

	// Backend info
	fmt.Println("Backend:")
	fmt.Printf("  File:      %s:%d\n", contract.Backend.File, contract.Backend.Line)
	fmt.Printf("  Framework: %s\n", contract.Backend.Framework)
	fmt.Printf("  Handler:   %s\n", contract.Backend.Handler)

	if contract.Backend.RequestSchema != nil {
		fmt.Println("  Request Schema:")
		printSchema(contract.Backend.RequestSchema, "    ")
	}
	if contract.Backend.ResponseSchema != nil {
		fmt.Println("  Response Schema:")
		printSchema(contract.Backend.ResponseSchema, "    ")
	}

	// Frontend calls
	fmt.Println()
	fmt.Printf("Frontend Calls: %d\n", len(contract.FrontendCalls))
	for i, call := range contract.FrontendCalls {
		if i >= 5 {
			fmt.Printf("  ... and %d more\n", len(contract.FrontendCalls)-5)
			break
		}
		fmt.Printf("  %s:%d (%s)\n", call.File, call.Line, call.CallType)
	}

	// Mismatches
	if len(contract.Mismatches) > 0 {
		fmt.Println()
		fmt.Printf("Mismatches: %d errors, %d warnings\n",
			contract.ErrorCount(), contract.WarningCount())
		for _, m := range contract.Mismatches {
			severity := "warning"
			if m.Severity == contracts.SeverityError {
				severity = "ERROR"
			}
			fmt.Printf("  [%s] %s: %s\n", severity, m.FieldPath, m.Description)
			if m.BackendType != "" && m.FrontendType != "" {
				fmt.Printf("    Backend: %s, Frontend: %s\n", m.BackendType, m.FrontendType)
			}
		}
	}

	fmt.Println()
	fmt.Printf("First seen:  %s\n", contract.FirstSeen.Format("2006-01-02 15:04"))
	fmt.Printf("Last seen:   %s\n", contract.LastSeen.Format("2006-01-02 15:04"))

	return nil
}

// RunContractsVerify marks a contract as verified.
func RunContractsVerify(args []string) error {
	fs := flag.NewFlagSet("contracts verify", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New(`usage: palace contracts verify <contract-id>

Mark a contract as verified (correct and intentional).

Arguments:
  <contract-id>    ID of contract to verify (e.g., ctr_abc123)

Examples:
  palace contracts verify ctr_abc123`)
	}

	return ExecuteContractsVerify(ContractsOptions{
		Root:       *root,
		ContractID: remaining[0],
	})
}

// ExecuteContractsVerify marks a contract as verified.
func ExecuteContractsVerify(opts ContractsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	store := contracts.NewStore(mem.DB())

	contract, err := store.GetContract(opts.ContractID)
	if err != nil {
		return fmt.Errorf("get contract: %w", err)
	}
	if contract == nil {
		return fmt.Errorf("contract not found: %s", opts.ContractID)
	}

	if contract.Status == contracts.ContractVerified {
		fmt.Printf("Contract %s is already verified.\n", opts.ContractID)
		return nil
	}

	// Show contract details
	fmt.Printf("Verifying contract: %s\n", opts.ContractID)
	fmt.Printf("Endpoint: %s %s\n", contract.Method, contract.Endpoint)

	// Clear mismatches if any
	if len(contract.Mismatches) > 0 {
		if err := store.ClearMismatches(opts.ContractID); err != nil {
			return fmt.Errorf("clear mismatches: %w", err)
		}
		fmt.Printf("Cleared %d mismatches\n", len(contract.Mismatches))
	}

	// Update status
	if err := store.UpdateStatus(opts.ContractID, contracts.ContractVerified); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	fmt.Printf("\n+ Verified contract: %s\n", opts.ContractID)
	return nil
}

// RunContractsIgnore ignores a contract.
func RunContractsIgnore(args []string) error {
	fs := flag.NewFlagSet("contracts ignore", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New(`usage: palace contracts ignore <contract-id>

Ignore a contract (won't be shown in future scans).

Arguments:
  <contract-id>    ID of contract to ignore (e.g., ctr_abc123)

Examples:
  palace contracts ignore ctr_abc123`)
	}

	return ExecuteContractsIgnore(ContractsOptions{
		Root:       *root,
		ContractID: remaining[0],
	})
}

// ExecuteContractsIgnore ignores a contract.
func ExecuteContractsIgnore(opts ContractsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	store := contracts.NewStore(mem.DB())

	contract, err := store.GetContract(opts.ContractID)
	if err != nil {
		return fmt.Errorf("get contract: %w", err)
	}
	if contract == nil {
		return fmt.Errorf("contract not found: %s", opts.ContractID)
	}

	if contract.Status == contracts.ContractIgnored {
		fmt.Printf("Contract %s is already ignored.\n", opts.ContractID)
		return nil
	}

	// Show contract details
	fmt.Printf("Ignoring contract: %s\n", opts.ContractID)
	fmt.Printf("Endpoint: %s %s\n", contract.Method, contract.Endpoint)

	// Update status
	if err := store.UpdateStatus(opts.ContractID, contracts.ContractIgnored); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	fmt.Printf("\nx Ignored contract: %s\n", opts.ContractID)
	return nil
}

// Helper functions for file collection and extraction

func collectBackendFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip common non-source directories
			name := d.Name()
			if name == "node_modules" || name == "vendor" || name == ".git" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		// Go, Python files
		if ext == ".go" || ext == ".py" {
			files = append(files, path)
		}
		// JavaScript/TypeScript for Express backends
		if ext == ".js" || ext == ".ts" {
			// Only include if likely backend (has route patterns)
			if isLikelyBackendJS(path) {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

func collectFrontendFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "vendor" || name == ".git" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		// Frontend TypeScript/JavaScript files
		if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func isLikelyBackendJS(path string) bool {
	// Check if the file path suggests backend
	lowerPath := strings.ToLower(path)
	backendPatterns := []string{
		"server", "api", "route", "controller", "handler",
		"backend", "express", "router", "endpoint",
	}
	for _, pattern := range backendPatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}
	return false
}

func extractEndpoints(files []string) ([]contracts.EndpointInput, error) {
	var endpoints []contracts.EndpointInput

	goExtractor := extractors.NewGoHTTPExtractor()
	expressExtractor := extractors.NewExpressExtractor()
	fastapiExtractor := extractors.NewFastAPIExtractor()

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		ext := filepath.Ext(file)
		var extracted []extractors.ExtractedEndpoint

		switch ext {
		case ".go":
			extracted, err = goExtractor.ExtractEndpointsFromContent(content, file)
		case ".js", ".ts":
			extracted, err = expressExtractor.ExtractEndpointsFromContent(content, file)
		case ".py":
			extracted, err = fastapiExtractor.ExtractEndpointsFromContent(content, file)
		}

		if err != nil {
			continue
		}

		for _, ep := range extracted {
			endpoints = append(endpoints, contracts.EndpointInput{
				Method:         ep.Method,
				Path:           ep.Path,
				File:           ep.File,
				Line:           ep.Line,
				Handler:        ep.Handler,
				Framework:      ep.Framework,
				RequestSchema:  ep.RequestSchema,
				ResponseSchema: ep.ResponseSchema,
			})
		}
	}

	return endpoints, nil
}

func extractCalls(files []string) ([]contracts.CallInput, error) {
	var calls []contracts.CallInput

	fetchExtractor := extractors.NewFetchExtractor()
	axiosExtractor := extractors.NewAxiosExtractor()

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Try fetch extractor
		fetchCalls, err := fetchExtractor.ExtractCallsFromContent(content, file)
		if err == nil {
			for _, call := range fetchCalls {
				calls = append(calls, contracts.CallInput{
					Method:    call.Method,
					URL:       call.URL,
					File:      call.File,
					Line:      call.Line,
					IsDynamic: call.IsDynamic,
				})
			}
		}

		// Try axios extractor
		axiosCalls, err := axiosExtractor.ExtractCallsFromContent(content, file)
		if err == nil {
			for _, call := range axiosCalls {
				calls = append(calls, contracts.CallInput{
					Method:    call.Method,
					URL:       call.URL,
					File:      call.File,
					Line:      call.Line,
					IsDynamic: call.IsDynamic,
				})
			}
		}
	}

	return calls, nil
}

func printSchema(schema *contracts.TypeSchema, indent string) {
	if schema == nil {
		return
	}

	switch schema.Type {
	case contracts.SchemaTypeObject:
		fmt.Printf("%s{\n", indent)
		for name, prop := range schema.Properties {
			required := ""
			if schema.IsRequired(name) {
				required = " (required)"
			}
			fmt.Printf("%s  %s: %s%s\n", indent, name, prop.Type, required)
		}
		fmt.Printf("%s}\n", indent)
	case contracts.SchemaTypeArray:
		if schema.Items != nil {
			fmt.Printf("%s[]%s\n", indent, schema.Items.Type)
		} else {
			fmt.Printf("%s[]\n", indent)
		}
	default:
		fmt.Printf("%s%s\n", indent, schema.Type)
	}
}
