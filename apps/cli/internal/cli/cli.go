// Package cli is the main entry point for the Mind Palace CLI application.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/commands"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/util"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/jsonc"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/update"
)

func init() {
	butler.SetJSONCDecoder(jsonc.DecodeFile)
}

// ============================================================================
// Validation Helpers - delegating to flags package
// ============================================================================

func validateConfidence(v float64) error { return flags.ValidateConfidence(v) }
func validateLimit(v int) error          { return flags.ValidateLimit(v) }
func validateScope(v string) error       { return flags.ValidateScope(v) }
func validatePort(v int) error           { return flags.ValidatePort(v) }

// Run executes the application given the command-line arguments.
func Run(args []string) error {
	if len(args) == 0 {
		return usage()
	}

	checkForUpdates(args)
	switch args[0] {
	// Core commands
	case "explore":
		return cmdExplore(args[1:])
	case "store":
		return cmdStore(args[1:])
	case "recall":
		return cmdRecall(args[1:])
	case "brief":
		return cmdBrief(args[1:])

	// Setup & Index
	case "init":
		return cmdInit(args[1:])
	case "scan":
		return cmdScan(args[1:])
	case "check":
		return cmdCheck(args[1:])
	case "stats":
		return cmdStats(args[1:])

	// Services
	case "serve":
		return cmdServe(args[1:])
	case "dashboard":
		return cmdDashboard(args[1:])

	// Agents & Sessions
	case "session":
		return cmdSession(args[1:])

	// Cross-workspace
	case "corridor":
		return cmdCorridor(args[1:])

	// Governance
	case "proposals":
		return cmdProposals(args[1:])
	case "approve":
		return cmdApprove(args[1:])
	case "reject":
		return cmdReject(args[1:])

	// Housekeeping
	case "clean":
		return cmdClean(args[1:])
	case "mcp-config":
		return cmdMCPConfig(args[1:])
	case "update":
		return cmdUpdate(args[1:])
	case "version", "--version", "-v":
		return cmdVersion(args[1:])
	case "help", "-h", "--help":
		if len(args) > 1 {
			return cmdHelp(args[1:])
		}
		return usage()

	default:
		// Check for shorthand: palace "content" -> palace store "content"
		if len(args) > 0 && !strings.HasPrefix(args[0], "-") && looksLikeContent(args[0]) {
			return cmdStore(args)
		}
		if len(args) > 0 {
			return fmt.Errorf("unknown command: %s\nRun 'palace help' for usage", args[0])
		}
		return fmt.Errorf("unknown command\nRun 'palace help' for usage")
	}
}

func usage() error {
	return commands.ShowUsage()
}

// ============================================================================
// Core Commands - delegating to commands package
// ============================================================================

// cmdExplore delegates to commands.RunExplore
func cmdExplore(args []string) error {
	return commands.RunExplore(args)
}

// cmdStore delegates to commands.RunStore
func cmdStore(args []string) error {
	return commands.RunStore(args)
}

// cmdRecall delegates to commands.RunRecall
func cmdRecall(args []string) error {
	return commands.RunRecall(args)
}

// cmdBrief delegates to commands.RunBrief
func cmdBrief(args []string) error {
	return commands.RunBrief(args)
}

// ============================================================================
// Setup & Index Commands - delegating to commands package
// ============================================================================

// cmdInit delegates to commands.RunInit (enter command)
func cmdInit(args []string) error {
	return commands.RunInit(args)
}

// cmdScan delegates to commands.RunScan
func cmdScan(args []string) error {
	return commands.RunScan(args)
}

// cmdCheck delegates to commands.RunCheck
func cmdCheck(args []string) error {
	return commands.RunCheck(args)
}

// cmdStats delegates to commands.RunStats
func cmdStats(args []string) error {
	return commands.RunStats(args)
}

// ============================================================================
// Service Commands - delegating to commands package
// ============================================================================

// cmdServe delegates to commands.RunServe
func cmdServe(args []string) error {
	return commands.RunServe(args)
}

// cmdDashboard delegates to commands.RunDashboard
func cmdDashboard(args []string) error {
	return commands.RunDashboard(args)
}

// ============================================================================
// Session & Corridor Commands - delegating to commands package
// ============================================================================

// cmdSession delegates to commands.RunSession
func cmdSession(args []string) error {
	return commands.RunSession(args)
}

// cmdCorridor delegates to commands.RunCorridor
func cmdCorridor(args []string) error {
	return commands.RunCorridor(args)
}

// ============================================================================
// Governance Commands - delegating to commands package
// ============================================================================

// cmdProposals delegates to commands.RunProposals
func cmdProposals(args []string) error {
	return commands.RunProposals(args)
}

// cmdApprove delegates to commands.RunApprove
func cmdApprove(args []string) error {
	return commands.RunApprove(args)
}

// cmdReject delegates to commands.RunReject
func cmdReject(args []string) error {
	return commands.RunReject(args)
}

// ============================================================================

// cmdClean delegates to commands.RunClean (maintenance command)
func cmdClean(args []string) error {
	return commands.RunClean(args)
}

// cmdMCPConfig delegates to commands.RunMCPConfig
func cmdMCPConfig(args []string) error {
	return commands.RunMCPConfig(args)
}

// cmdUpdate delegates to commands.RunUpdate
func cmdUpdate(args []string) error {
	commands.BuildVersion = GetVersion()
	return commands.RunUpdate(args)
}

// ============================================================================
// Help Command - delegating to commands package
// ============================================================================

// cmdHelp delegates to commands.RunHelp
func cmdHelp(args []string) error {
	return commands.RunHelp(args)
}

// ============================================================================
// Utility Functions
// ============================================================================

// looksLikeContent checks if a string looks like content rather than a command.
// Used for shorthand: palace "Let's use JWT" -> palace store "Let's use JWT"
func looksLikeContent(s string) bool {
	// If it contains spaces or looks like a sentence, it's content
	if strings.Contains(s, " ") {
		return true
	}
	// Check for sentence-like patterns
	lowerS := strings.ToLower(s)
	contentSignals := []string{"let's", "we", "what", "how", "til", "note:", "idea:", "decision:"}
	for _, sig := range contentSignals {
		if strings.HasPrefix(lowerS, sig) {
			return true
		}
	}
	// Check if it ends with punctuation
	if strings.HasSuffix(s, "?") || strings.HasSuffix(s, "!") || strings.HasSuffix(s, ".") {
		return true
	}
	return false
}

func checkForUpdates(args []string) {
	if len(args) == 0 {
		return
	}
	cmd := args[0]
	if cmd == "version" || cmd == "--version" || cmd == "-v" || cmd == "update" {
		return
	}

	cacheDir, err := update.GetCacheDir()
	if err != nil {
		return
	}

	result, err := update.CheckCached(GetVersion(), cacheDir)
	if err != nil {
		return
	}

	if result.UpdateAvailable {
		fmt.Fprintf(os.Stderr, "Update available: v%s -> v%s (run 'palace update')\n\n", result.CurrentVersion, result.LatestVersion)
	}
}

// boolFlag wraps flags.BoolFlag for backward compatibility
type boolFlag = flags.BoolFlag

// mustAbs delegates to util.MustAbs
func mustAbs(p string) string { return util.MustAbs(p) }

// scopeFileCount delegates to util.ScopeFileCount
func scopeFileCount(cp model.ContextPack) int { return util.ScopeFileCount(cp) }

// truncateLine delegates to util.TruncateLine
func truncateLine(s string, maxLen int) string { return util.TruncateLine(s, maxLen) }
