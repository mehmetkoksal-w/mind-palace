package commands

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/util"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "proposals",
		Aliases:     []string{"props"},
		Description: "List pending proposals awaiting approval",
		Run:         RunProposals,
	})
	Register(&Command{
		Name:        "approve",
		Aliases:     []string{},
		Description: "Approve a pending proposal",
		Run:         RunApprove,
	})
	Register(&Command{
		Name:        "reject",
		Aliases:     []string{},
		Description: "Reject a pending proposal",
		Run:         RunReject,
	})
}

// ProposalsOptions contains the configuration for the proposals command.
type ProposalsOptions struct {
	Root       string
	Status     string // pending, approved, rejected, expired, or empty for all
	ProposedAs string // decision, learning, or empty for all
	Limit      int
}

// RunProposals executes the proposals command.
func RunProposals(args []string) error {
	fs := flag.NewFlagSet("proposals", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	status := fs.String("status", "pending", "filter by status: pending, approved, rejected, expired, all")
	proposedAs := fs.String("type", "", "filter by type: decision, learning")
	limit := fs.Int("limit", 20, "maximum number of proposals to show")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Handle "all" status
	statusFilter := *status
	if statusFilter == "all" {
		statusFilter = ""
	}

	return ExecuteProposals(ProposalsOptions{
		Root:       *root,
		Status:     statusFilter,
		ProposedAs: *proposedAs,
		Limit:      *limit,
	})
}

// ExecuteProposals lists proposals matching the criteria.
func ExecuteProposals(opts ProposalsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	proposals, err := mem.GetProposals(opts.Status, opts.ProposedAs, opts.Limit)
	if err != nil {
		return fmt.Errorf("get proposals: %w", err)
	}

	if len(proposals) == 0 {
		if opts.Status == memory.ProposalStatusPending || opts.Status == "" {
			fmt.Println("No pending proposals.")
		} else {
			fmt.Printf("No proposals with status '%s'.\n", opts.Status)
		}
		return nil
	}

	// Get count for header
	pendingCount, _ := mem.GetPendingProposalsCount()

	fmt.Printf("Proposals (%d pending)\n", pendingCount)
	fmt.Println(strings.Repeat("=", 60))

	for _, p := range proposals {
		statusIcon := "?"
		switch p.Status {
		case memory.ProposalStatusPending:
			statusIcon = "?"
		case memory.ProposalStatusApproved:
			statusIcon = "+"
		case memory.ProposalStatusRejected:
			statusIcon = "x"
		case memory.ProposalStatusExpired:
			statusIcon = "-"
		}

		typeIcon := "D"
		if p.ProposedAs == memory.ProposedAsLearning {
			typeIcon = "L"
		}

		fmt.Printf("\n[%s] %s %s\n", statusIcon, typeIcon, p.ID)
		fmt.Printf("    Type: %s | Status: %s\n", p.ProposedAs, p.Status)
		scopeInfo := p.Scope
		if p.ScopePath != "" {
			scopeInfo = fmt.Sprintf("%s:%s", p.Scope, p.ScopePath)
		}
		fmt.Printf("    Scope: %s\n", scopeInfo)
		fmt.Printf("    Content: %s\n", util.TruncateLine(p.Content, 60))
		if p.ClassificationConfidence > 0 {
			fmt.Printf("    Confidence: %.0f%%\n", p.ClassificationConfidence*100)
		}
		fmt.Printf("    Created: %s\n", p.CreatedAt.Format("2006-01-02 15:04"))
		if p.PromotedToID != "" {
			fmt.Printf("    Promoted to: %s\n", p.PromotedToID)
		}
		if p.ReviewNote != "" {
			fmt.Printf("    Review note: %s\n", p.ReviewNote)
		}
	}

	fmt.Println()
	if opts.Status == memory.ProposalStatusPending || opts.Status == "" {
		fmt.Println("Use 'palace approve <id>' to approve a proposal")
		fmt.Println("Use 'palace reject <id> --note \"reason\"' to reject a proposal")
	}

	return nil
}

// ApproveOptions contains the configuration for the approve command.
type ApproveOptions struct {
	Root       string
	ProposalID string
	ReviewedBy string
	ReviewNote string
}

// RunApprove executes the approve command.
func RunApprove(args []string) error {
	fs := flag.NewFlagSet("approve", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	reviewedBy := fs.String("by", "cli", "reviewer identifier")
	reviewNote := fs.String("note", "", "optional review note")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New(`usage: palace approve <proposal-id> [options]

Approve a pending proposal, creating the corresponding decision or learning.

Arguments:
  <proposal-id>    ID of the proposal to approve (e.g., prop_abc123)

Options:
  --by <name>      Reviewer identifier (default: cli)
  --note <text>    Optional review note

Examples:
  palace approve prop_abc123
  palace approve prop_abc123 --note "Looks good"
  palace approve prop_abc123 --by "john"`)
	}

	return ExecuteApprove(ApproveOptions{
		Root:       *root,
		ProposalID: remaining[0],
		ReviewedBy: *reviewedBy,
		ReviewNote: *reviewNote,
	})
}

// ExecuteApprove approves a proposal.
func ExecuteApprove(opts ApproveOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	// Get the proposal first to show what we're approving
	proposal, err := mem.GetProposal(opts.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	// Show proposal details
	fmt.Printf("Approving proposal: %s\n", opts.ProposalID)
	fmt.Printf("Type: %s\n", proposal.ProposedAs)
	fmt.Printf("Content: %s\n", util.TruncateLine(proposal.Content, 80))
	fmt.Println()

	// Approve it
	promotedID, err := mem.ApproveProposal(opts.ProposalID, opts.ReviewedBy, opts.ReviewNote)
	if err != nil {
		return fmt.Errorf("approve proposal: %w", err)
	}

	// Audit log for approval
	_, _ = mem.AddAuditLog(memory.AuditLogEntry{
		Action:     memory.AuditActionApprove,
		ActorType:  memory.AuditActorHuman,
		ActorID:    opts.ReviewedBy,
		TargetID:   opts.ProposalID,
		TargetKind: "proposal",
		Details:    fmt.Sprintf(`{"promoted_to": "%s", "note": "%s"}`, promotedID, opts.ReviewNote),
	})

	// Output success
	fmt.Printf("+ Approved! Created %s: %s\n", proposal.ProposedAs, promotedID)
	if opts.ReviewNote != "" {
		fmt.Printf("  Note: %s\n", opts.ReviewNote)
	}

	return nil
}

// RejectOptions contains the configuration for the reject command.
type RejectOptions struct {
	Root       string
	ProposalID string
	ReviewedBy string
	ReviewNote string
}

// RunReject executes the reject command.
func RunReject(args []string) error {
	fs := flag.NewFlagSet("reject", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	reviewedBy := fs.String("by", "cli", "reviewer identifier")
	reviewNote := fs.String("note", "", "reason for rejection (recommended)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New(`usage: palace reject <proposal-id> [options]

Reject a pending proposal.

Arguments:
  <proposal-id>    ID of the proposal to reject (e.g., prop_abc123)

Options:
  --by <name>      Reviewer identifier (default: cli)
  --note <text>    Reason for rejection (recommended)

Examples:
  palace reject prop_abc123 --note "Not accurate"
  palace reject prop_abc123 --note "Duplicate of existing decision d_xyz"`)
	}

	return ExecuteReject(RejectOptions{
		Root:       *root,
		ProposalID: remaining[0],
		ReviewedBy: *reviewedBy,
		ReviewNote: *reviewNote,
	})
}

// ExecuteReject rejects a proposal.
func ExecuteReject(opts RejectOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	// Get the proposal first to show what we're rejecting
	proposal, err := mem.GetProposal(opts.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	// Show proposal details
	fmt.Printf("Rejecting proposal: %s\n", opts.ProposalID)
	fmt.Printf("Type: %s\n", proposal.ProposedAs)
	fmt.Printf("Content: %s\n", util.TruncateLine(proposal.Content, 80))
	fmt.Println()

	// Reject it
	err = mem.RejectProposal(opts.ProposalID, opts.ReviewedBy, opts.ReviewNote)
	if err != nil {
		return fmt.Errorf("reject proposal: %w", err)
	}

	// Audit log for rejection
	_, _ = mem.AddAuditLog(memory.AuditLogEntry{
		Action:     memory.AuditActionReject,
		ActorType:  memory.AuditActorHuman,
		ActorID:    opts.ReviewedBy,
		TargetID:   opts.ProposalID,
		TargetKind: "proposal",
		Details:    fmt.Sprintf(`{"note": "%s"}`, opts.ReviewNote),
	})

	// Output success
	fmt.Printf("x Rejected proposal: %s\n", opts.ProposalID)
	if opts.ReviewNote != "" {
		fmt.Printf("  Reason: %s\n", opts.ReviewNote)
	}

	return nil
}
