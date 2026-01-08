package commands

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/util"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "recall",
		Description: "Retrieve knowledge from the palace",
		Run:         RunRecall,
	})
}

// RunRecall executes the recall command.
func RunRecall(args []string) error {
	if len(args) == 0 {
		return runRecallList([]string{})
	}

	// Check for subcommands
	switch args[0] {
	case "update":
		return runRecallUpdate(args[1:])
	case "link":
		return runRecallLink(args[1:])
	default:
		// Not a subcommand, treat as list with potential query
		return runRecallList(args)
	}
}

// runRecallList lists/searches knowledge.
func runRecallList(args []string) error {
	fs := flag.NewFlagSet("recall", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	scope := fs.String("scope", "", "filter by scope (file, room, palace)")
	path := fs.String("path", "", "filter by scope path")
	limit := fs.Int("limit", 10, "maximum results")
	typeFilter := fs.String("type", "", "filter by type: decision, idea, learning")
	pending := fs.Bool("pending", false, "show decisions awaiting outcome")
	since := fs.Int("since", 30, "for --pending: show decisions older than N days")
	all := fs.Bool("all", false, "for --pending: show all pending regardless of age")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate inputs
	if err := flags.ValidateLimit(*limit); err != nil {
		return err
	}
	if *scope != "" {
		if err := flags.ValidateScope(*scope); err != nil {
			return err
		}
	}
	if *typeFilter != "" && *typeFilter != "decision" && *typeFilter != "idea" && *typeFilter != "learning" {
		return fmt.Errorf("invalid --type %q; must be decision, idea, or learning", *typeFilter)
	}

	remaining := fs.Args()
	query := ""
	if len(remaining) > 0 {
		query = strings.Join(remaining, " ")
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	// Handle --pending flag (decisions awaiting outcome)
	if *pending {
		return recallPending(mem, *since, *all, *limit)
	}

	// Handle --type filter or default to learnings
	switch *typeFilter {
	case "decision":
		return recallDecisions(mem, query, *scope, *path, *limit)
	case "idea":
		return recallIdeas(mem, query, *scope, *path, *limit)
	case "learning", "":
		return recallLearnings(mem, query, *scope, *path, *limit)
	}

	return nil
}

// recallLearnings retrieves learnings.
func recallLearnings(mem *memory.Memory, query, scope, scopePath string, limit int) error {
	var learnings []memory.Learning
	var err error

	if query != "" {
		learnings, err = mem.SearchLearnings(query, limit)
	} else {
		learnings, err = mem.GetLearnings(scope, scopePath, limit)
	}
	if err != nil {
		return fmt.Errorf("recall learnings: %w", err)
	}

	if len(learnings) == 0 {
		fmt.Println("No learnings found.")
		return nil
	}

	fmt.Printf("\n📝 Learnings\n")
	fmt.Println(strings.Repeat("─", 60))

	for i := range learnings {
		l := &learnings[i]
		confidenceBar := strings.Repeat("█", int(l.Confidence*10)) + strings.Repeat("░", 10-int(l.Confidence*10))
		scopeInfo := l.Scope
		if l.ScopePath != "" {
			scopeInfo = fmt.Sprintf("%s:%s", l.Scope, l.ScopePath)
		}
		fmt.Printf("\n[%s] %s (%.0f%%)\n", l.ID, confidenceBar, l.Confidence*100)
		fmt.Printf("  Scope:  %s\n", scopeInfo)
		fmt.Printf("  Source: %s | Used: %d times\n", l.Source, l.UseCount)
		fmt.Printf("  %s\n", l.Content)
	}
	fmt.Println()

	return nil
}

// recallDecisions retrieves decisions.
func recallDecisions(mem *memory.Memory, query, scope, scopePath string, limit int) error {
	var decisions []memory.Decision
	var err error

	if query != "" {
		decisions, err = mem.SearchDecisions(query, limit)
	} else {
		decisions, err = mem.GetDecisions("", "", scope, scopePath, limit)
	}
	if err != nil {
		return fmt.Errorf("recall decisions: %w", err)
	}

	if len(decisions) == 0 {
		fmt.Println("No decisions found.")
		return nil
	}

	fmt.Printf("\n🔨 Decisions\n")
	fmt.Println(strings.Repeat("─", 60))

	for i := range decisions {
		d := &decisions[i]
		statusIcon := "🔵"
		switch d.Status {
		case memory.DecisionStatusSuperseded:
			statusIcon = "🔄"
		case memory.DecisionStatusReversed:
			statusIcon = "↩️"
		}

		outcomeIcon := "❓"
		switch d.Outcome {
		case memory.DecisionOutcomeSuccessful:
			outcomeIcon = "✅"
		case memory.DecisionOutcomeFailed:
			outcomeIcon = "❌"
		case memory.DecisionOutcomeMixed:
			outcomeIcon = "⚖️"
		}

		scopeInfo := d.Scope
		if d.ScopePath != "" {
			scopeInfo = fmt.Sprintf("%s:%s", d.Scope, d.ScopePath)
		}

		fmt.Printf("\n%s [%s] %s\n", statusIcon, d.ID, outcomeIcon)
		fmt.Printf("  Scope:  %s\n", scopeInfo)
		fmt.Printf("  %s\n", util.TruncateLine(d.Content, 60))
		if d.Rationale != "" {
			fmt.Printf("  Rationale: %s\n", util.TruncateLine(d.Rationale, 50))
		}
	}
	fmt.Println()

	return nil
}

// recallIdeas retrieves ideas.
func recallIdeas(mem *memory.Memory, query, scope, scopePath string, limit int) error {
	var ideas []memory.Idea
	var err error

	if query != "" {
		ideas, err = mem.SearchIdeas(query, limit)
	} else {
		ideas, err = mem.GetIdeas("", scope, scopePath, limit)
	}
	if err != nil {
		return fmt.Errorf("recall ideas: %w", err)
	}

	if len(ideas) == 0 {
		fmt.Println("No ideas found.")
		return nil
	}

	fmt.Printf("\n💡 Ideas\n")
	fmt.Println(strings.Repeat("─", 60))

	for j := range ideas {
		i := &ideas[j]
		statusIcon := "💡"
		switch i.Status {
		case memory.IdeaStatusExploring:
			statusIcon = "🔍"
		case memory.IdeaStatusImplemented:
			statusIcon = "✅"
		case memory.IdeaStatusDropped:
			statusIcon = "❌"
		}

		scopeInfo := i.Scope
		if i.ScopePath != "" {
			scopeInfo = fmt.Sprintf("%s:%s", i.Scope, i.ScopePath)
		}

		fmt.Printf("\n%s [%s] %s\n", statusIcon, i.ID, i.Status)
		fmt.Printf("  Scope:  %s\n", scopeInfo)
		fmt.Printf("  %s\n", util.TruncateLine(i.Content, 60))
		if i.Context != "" {
			fmt.Printf("  Context: %s\n", util.TruncateLine(i.Context, 50))
		}
	}
	fmt.Println()

	return nil
}

// recallPending shows decisions awaiting outcome.
func recallPending(mem *memory.Memory, since int, all bool, limit int) error {
	var decisions []memory.Decision
	var err error

	if all {
		decisions, err = mem.GetDecisionsAwaitingReview(0, limit)
	} else {
		cutoff := time.Now().UTC().AddDate(0, 0, -since)
		decisions, err = mem.GetDecisionsSince(cutoff, limit)
		// Filter to only those with unknown outcome
		var filtered []memory.Decision
		for k := range decisions {
			d := &decisions[k]
			if d.Outcome == memory.DecisionOutcomeUnknown {
				filtered = append(filtered, *d)
			}
		}
		decisions = filtered
	}
	if err != nil {
		return fmt.Errorf("get decisions: %w", err)
	}

	if len(decisions) == 0 {
		fmt.Println("No decisions awaiting review.")
		return nil
	}

	fmt.Printf("\n📋 Decisions Awaiting Review\n")
	fmt.Println(strings.Repeat("─", 60))

	for m := range decisions {
		d := &decisions[m]
		statusIcon := "🔵"
		switch d.Status {
		case memory.DecisionStatusSuperseded:
			statusIcon = "🔄"
		case memory.DecisionStatusReversed:
			statusIcon = "↩️"
		}

		age := time.Since(d.CreatedAt)
		ageStr := fmt.Sprintf("%.0f days", age.Hours()/24)
		if age.Hours() < 24 {
			ageStr = fmt.Sprintf("%.0f hours", age.Hours())
		}

		fmt.Printf("\n%s [%s] (%s ago)\n", statusIcon, d.ID, ageStr)
		fmt.Printf("  %s\n", util.TruncateLine(d.Content, 60))
		if d.Rationale != "" {
			fmt.Printf("  Rationale: %s\n", util.TruncateLine(d.Rationale, 50))
		}
		fmt.Printf("  → Use: palace recall update %s <success|failed|mixed>\n", d.ID)
	}
	fmt.Println()

	return nil
}

// runRecallUpdate records the outcome of a decision.
func runRecallUpdate(args []string) error {
	fs := flag.NewFlagSet("recall update", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	note := fs.String("note", "", "optional note about the outcome")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) < 2 {
		return errors.New(`usage: palace recall update <decision-id> <outcome> [--note "..."]

Outcomes:
  success   The decision worked out well
  failed    The decision didn't work out
  mixed     The decision had mixed results

Examples:
  palace recall update d_abc123 success
  palace recall update d_abc123 failed --note "Caused performance issues"
  palace recall update d_abc123 mixed --note "Worked for API, not for CLI"`)
	}

	decisionID := remaining[0]
	outcome := remaining[1]

	// Normalize outcome names
	outcomeMap := map[string]string{
		"success":    "successful",
		"successful": "successful",
		"failed":     "failed",
		"fail":       "failed",
		"mixed":      "mixed",
	}
	normalizedOutcome, ok := outcomeMap[outcome]
	if !ok {
		return fmt.Errorf("invalid outcome %q; must be success, failed, or mixed", outcome)
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	if err := mem.RecordDecisionOutcome(decisionID, normalizedOutcome, *note); err != nil {
		return fmt.Errorf("record outcome: %w", err)
	}

	outcomeIcon := "✅"
	switch normalizedOutcome {
	case "failed":
		outcomeIcon = "❌"
	case "mixed":
		outcomeIcon = "⚖️"
	}

	fmt.Printf("%s Outcome recorded for %s: %s\n", outcomeIcon, decisionID, normalizedOutcome)
	if *note != "" {
		fmt.Printf("  Note: %s\n", *note)
	}
	return nil
}

// runRecallLink manages links between records.
func runRecallLink(args []string) error {
	fs := flag.NewFlagSet("recall link", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")

	// Relation flags
	supersedes := fs.String("supersedes", "", "link supersedes this decision ID")
	implements := fs.String("implements", "", "link implements code target (file:line-range)")
	supports := fs.String("supports", "", "link supports this record ID")
	contradicts := fs.String("contradicts", "", "link contradicts this record ID")
	inspiredBy := fs.String("inspired-by", "", "link was inspired by this record ID")
	related := fs.String("related", "", "link is related to this record ID")

	// List flags
	listSource := fs.Bool("from", false, "list links from this source ID")
	listTarget := fs.Bool("to", false, "list links to this target ID")
	listAll := fs.Bool("all", false, "list all links for this ID (source or target)")
	listStale := fs.Bool("stale", false, "list all stale code links")

	// Delete flag
	deleteID := fs.String("delete", "", "delete a link by ID")

	if err := fs.Parse(args); err != nil {
		return err
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	// Handle delete
	if *deleteID != "" {
		if err := mem.DeleteLink(*deleteID); err != nil {
			return fmt.Errorf("delete link: %w", err)
		}
		fmt.Printf("Deleted link: %s\n", *deleteID)
		return nil
	}

	// Handle list stale
	if *listStale {
		links, err := mem.GetStaleLinks()
		if err != nil {
			return fmt.Errorf("get stale links: %w", err)
		}
		if len(links) == 0 {
			fmt.Println("No stale links found.")
			return nil
		}
		fmt.Printf("\n⚠️  Stale Links (%d found)\n", len(links))
		fmt.Println(strings.Repeat("─", 60))
		for i := range links {
			l := &links[i]
			fmt.Printf("\n[%s] %s → %s\n", l.ID, l.SourceID, l.TargetID)
			fmt.Printf("  Relation: %s\n", l.Relation)
			fmt.Printf("  Created: %s\n", l.CreatedAt.Format("2006-01-02 15:04"))
		}
		return nil
	}

	remaining := fs.Args()

	// Handle list operations
	if *listSource || *listTarget || *listAll {
		if len(remaining) == 0 {
			return errors.New("usage: palace recall link --from|--to|--all <record-id>")
		}
		recordID := remaining[0]

		var links []memory.Link
		switch {
		case *listAll:
			links, err = mem.GetAllLinksFor(recordID)
		case *listSource:
			links, err = mem.GetLinksForSource(recordID)
		default:
			links, err = mem.GetLinksForTarget(recordID)
		}
		if err != nil {
			return fmt.Errorf("get links: %w", err)
		}

		if len(links) == 0 {
			fmt.Printf("No links found for %s\n", recordID)
			return nil
		}

		fmt.Printf("\n🔗 Links for %s (%d found)\n", recordID, len(links))
		fmt.Println(strings.Repeat("─", 60))
		for i := range links {
			l := &links[i]
			direction := "→"
			other := l.TargetID
			if l.TargetID == recordID {
				direction = "←"
				other = l.SourceID
			}
			staleIndicator := ""
			if l.IsStale {
				staleIndicator = " ⚠️ stale"
			}
			fmt.Printf("\n[%s] %s %s (%s)%s\n", l.ID, direction, other, l.Relation, staleIndicator)
			fmt.Printf("  %s %s %s\n", l.SourceKind, l.Relation, l.TargetKind)
		}
		return nil
	}

	// Create a new link - need source ID and a relation flag
	if len(remaining) == 0 {
		return errors.New(`usage: palace recall link --<relation> <target> <source-id>

Create links between records (flags must come before the source ID):
  palace recall link --supersedes d_xyz789 d_abc123     # New decision supersedes old
  palace recall link --implements auth/jwt.go:15-45 i_abc123  # Idea implemented in code
  palace recall link --supports d_xyz789 i_abc123       # Idea supports decision
  palace recall link --contradicts d_xyz789 i_abc123    # Idea contradicts decision
  palace recall link --inspired-by i_other i_abc123     # Idea inspired by another
  palace recall link --related l_xyz i_abc123           # General relationship

List links:
  palace recall link --from <id>       # Links where ID is source
  palace recall link --to <id>         # Links where ID is target
  palace recall link --all <id>        # All links for ID
  palace recall link --stale           # List stale code links

Delete links:
  palace recall link --delete <link-id>`)
	}

	sourceID := remaining[0]

	// Determine which relation was specified
	var relation, targetID, targetKind string
	switch {
	case *supersedes != "":
		relation = memory.RelationSupersedes
		targetID = *supersedes
		targetKind = memory.TargetKindDecision
	case *implements != "":
		relation = memory.RelationImplements
		targetID = *implements
		targetKind = memory.TargetKindCode
	case *supports != "":
		relation = memory.RelationSupports
		targetID = *supports
		targetKind = inferTargetKind(*supports)
	case *contradicts != "":
		relation = memory.RelationContradicts
		targetID = *contradicts
		targetKind = inferTargetKind(*contradicts)
	case *inspiredBy != "":
		relation = memory.RelationInspiredBy
		targetID = *inspiredBy
		targetKind = inferTargetKind(*inspiredBy)
	case *related != "":
		relation = memory.RelationRelated
		targetID = *related
		targetKind = inferTargetKind(*related)
	default:
		return errors.New("must specify a relation: --supersedes, --implements, --supports, --contradicts, --inspired-by, or --related")
	}

	// Infer source kind from ID prefix
	sourceKind := inferTargetKind(sourceID)

	// Build the link
	link := memory.Link{
		SourceID:   sourceID,
		SourceKind: sourceKind,
		TargetID:   targetID,
		TargetKind: targetKind,
		Relation:   relation,
	}

	// For code links, validate the target and get mtime
	if targetKind == memory.TargetKindCode {
		parsed, mtime, err := memory.ValidateCodeTarget(rootPath, targetID)
		if err != nil {
			return fmt.Errorf("invalid code target: %w", err)
		}
		link.TargetMtime = mtime
		_ = parsed // validated
	}

	id, err := mem.AddLink(link)
	if err != nil {
		return fmt.Errorf("add link: %w", err)
	}

	fmt.Printf("🔗 Link created: %s\n", id)
	fmt.Printf("  %s %s → %s\n", sourceID, relation, targetID)
	return nil
}

// inferTargetKind infers the target kind from its ID prefix.
func inferTargetKind(id string) string {
	if strings.HasPrefix(id, "d_") {
		return memory.TargetKindDecision
	}
	if strings.HasPrefix(id, "i_") {
		return memory.TargetKindIdea
	}
	if strings.HasPrefix(id, "l_") {
		return memory.TargetKindLearning
	}
	if strings.Contains(id, ":") || strings.HasSuffix(id, ".go") ||
		strings.HasSuffix(id, ".ts") || strings.HasSuffix(id, ".py") {
		return memory.TargetKindCode
	}
	return "unknown"
}
