package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/util"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "store",
		Aliases:     []string{"remember", "learn"},
		Description: "Store knowledge in the palace (idea, decision, or learning)",
		Run:         RunStore,
	})
}

// StoreOptions contains the configuration for the store command.
type StoreOptions struct {
	Root       string
	Content    string
	Scope      string
	ScopePath  string
	Tags       []string
	AsType     string  // Force type: decision, idea, learning
	Confidence float64 // For learnings
	Direct     bool    // Direct write bypass (human-only)
}

// RunStore executes the store command with parsed arguments.
func RunStore(args []string) error {
	// Extract content (non-flag args) and flags separately
	var contentParts []string
	var flagArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				if arg == "--scope" || arg == "-scope" ||
					arg == "--path" || arg == "-path" ||
					arg == "--root" || arg == "-root" || arg == "-r" ||
					arg == "--tag" || arg == "-tag" ||
					arg == "--as" || arg == "-as" ||
					arg == "--confidence" || arg == "-confidence" {
					i++
					flagArgs = append(flagArgs, args[i])
				}
			}
		} else {
			contentParts = append(contentParts, arg)
		}
	}

	fs := flag.NewFlagSet("store", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	scope := fs.String("scope", "palace", "scope (file, room, palace)")
	path := fs.String("path", "", "scope path (file path or room name)")
	tag := fs.String("tag", "", "comma-separated tags")
	asType := fs.String("as", "", "force type: decision, idea, or learning")
	confidence := fs.Float64("confidence", 0.5, "confidence for learnings (0.0-1.0)")
	direct := fs.Bool("direct", false, "direct write (bypass proposals) for decisions/learnings; audited, human-only")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	// Validate inputs
	if err := flags.ValidateScope(*scope); err != nil {
		return err
	}
	if *asType != "" && *asType != "decision" && *asType != "idea" && *asType != "learning" {
		return fmt.Errorf("invalid --as type %q; must be decision, idea, or learning", *asType)
	}
	if *confidence < 0 || *confidence > 1 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0")
	}

	if len(contentParts) == 0 {
		return errors.New(`usage: palace store "content" [options]

Store knowledge in the palace. Content is auto-classified as an idea,
decision, or learning based on natural language signals.

Options:
  --scope <scope>       Scope: file, room, palace (default: palace)
  --path <path>         Scope path (file path or room name)
  --tag <tags>          Comma-separated tags
  --as <type>           Force type: decision, idea, or learning
  --confidence <n>      Confidence for learnings, 0.0-1.0 (default: 0.5)
	--direct              Direct write (bypass proposals) for decisions/learnings; audited, human-only

Examples:
  palace store "Let's use JWT for authentication"     # Auto-classified as decision
  palace store "What if we add caching?"              # Auto-classified as idea
  palace store "Always run tests before committing"   # Auto-classified as learning
  palace store "Use JWT" --as decision                # Force as decision
  palace store "Config is in /etc" --as learning --confidence 0.9`)
	}
	content := strings.Join(contentParts, " ")

	// Parse tags
	var tags []string
	if *tag != "" {
		for _, t := range strings.Split(*tag, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	return ExecuteStore(StoreOptions{
		Root:       *root,
		Content:    content,
		Scope:      *scope,
		ScopePath:  *path,
		Tags:       tags,
		AsType:     *asType,
		Confidence: *confidence,
		Direct:     *direct,
	})
}

// ExecuteStore stores knowledge with auto-classification.
func ExecuteStore(opts StoreOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	// Determine kind (explicit --as or auto-classify)
	var kind memory.RecordKind
	var classification memory.Classification

	if opts.AsType != "" {
		// Explicit type
		switch opts.AsType {
		case "decision":
			kind = memory.RecordKindDecision
		case "idea":
			kind = memory.RecordKindIdea
		case "learning":
			kind = memory.RecordKindLearning
		}
		classification = memory.Classification{Kind: kind, Confidence: 1.0, Signals: []string{"explicit"}}
	} else {
		// Auto-classify
		classification = memory.Classify(opts.Content)
		kind = classification.Kind

		// If low confidence, inform user
		if classification.NeedsConfirmation() {
			fmt.Printf("Auto-classified as %s (%.0f%% confidence)\n", kind, classification.Confidence*100)
			fmt.Printf("Signals: %v\n", classification.Signals)
			fmt.Println("Use --as decision, --as idea, or --as learning to override.")
		}
	}

	// Extract additional tags from content
	extractedTags := memory.ExtractTags(opts.Content)
	opts.Tags = append(opts.Tags, extractedTags...)

	// Store based on kind
	// Governance: decisions/learnings default to proposals unless --direct
	var id string
	switch kind {
	case memory.RecordKindIdea:
		idea := memory.Idea{
			Content:   opts.Content,
			Scope:     opts.Scope,
			ScopePath: opts.ScopePath,
			Source:    "cli",
		}
		id, err = mem.AddIdea(idea)
	case memory.RecordKindDecision, memory.RecordKindLearning:
		if opts.Direct {
			// Direct write path (audited)
			if kind == memory.RecordKindDecision {
				dec := memory.Decision{
					Content:   opts.Content,
					Scope:     opts.Scope,
					ScopePath: opts.ScopePath,
					Source:    "cli",
					Authority: string(memory.AuthorityApproved),
				}
				id, err = mem.AddDecision(dec)
			} else {
				learn := memory.Learning{
					Content:    opts.Content,
					Scope:      opts.Scope,
					ScopePath:  opts.ScopePath,
					Source:     "cli",
					Confidence: opts.Confidence,
					Authority:  string(memory.AuthorityApproved),
				}
				id, err = mem.AddLearning(learn)
			}
			if err != nil {
				return fmt.Errorf("store %s: %w", kind, err)
			}
			// Audit direct write
			targetKind := "decision"
			if kind == memory.RecordKindLearning {
				targetKind = "learning"
			}
			hash := sha256.Sum256([]byte(opts.Content))
			contentHash := hex.EncodeToString(hash[:])
			_, _ = mem.AddAuditLog(memory.AuditLogEntry{
				Action:     memory.AuditActionDirectWrite,
				ActorType:  memory.AuditActorHuman,
				ActorID:    "cli",
				TargetID:   id,
				TargetKind: targetKind,
				Details:    fmt.Sprintf(`{"scope":"%s","scope_path":"%s","content_hash":"%s"}`, opts.Scope, opts.ScopePath, contentHash),
			})
		} else {
			// Proposal path (default)
			proposedAs := memory.ProposedAsDecision
			if kind == memory.RecordKindLearning {
				proposedAs = memory.ProposedAsLearning
			}

			// Build classification signals JSON if auto-classified
			signalsJSON := "[]"
			if opts.AsType == "" && len(classification.Signals) > 0 {
				if data, mErr := json.Marshal(classification.Signals); mErr == nil {
					signalsJSON = string(data)
				}
			}

			prop := memory.Proposal{
				ProposedAs:               proposedAs,
				Content:                  opts.Content,
				Scope:                    opts.Scope,
				ScopePath:                opts.ScopePath,
				Source:                   "cli",
				ClassificationConfidence: classification.Confidence,
				ClassificationSignals:    signalsJSON,
			}

			// Dedupe check
			dedupeKey := memory.GenerateDedupeKey(prop.ProposedAs, prop.Content, prop.Scope, prop.ScopePath)
			if existing, _ := mem.CheckDuplicateProposal(dedupeKey); existing != nil {
				return fmt.Errorf("duplicate proposal already exists: %s", existing.ID)
			}
			prop.DedupeKey = dedupeKey

			id, err = mem.AddProposal(prop)
		}
	}

	if err != nil {
		return fmt.Errorf("store %s: %w", kind, err)
	}

	// Set tags if any
	if len(opts.Tags) > 0 {
		if err := mem.SetTags(id, string(kind), opts.Tags); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to set tags: %v\n", err)
		}
	}

	// Output
	if kind == memory.RecordKindIdea || opts.Direct {
		kindIcon := "💡"
		switch kind {
		case memory.RecordKindDecision:
			kindIcon = "🔨"
		case memory.RecordKindLearning:
			kindIcon = "📝"
		}
		fmt.Printf("%s Stored as %s: %s\n", kindIcon, kind, id)
		if opts.AsType == "" {
			fmt.Printf("  Classification: %.0f%% confidence\n", classification.Confidence*100)
		}
		fmt.Printf("  Scope: %s", opts.Scope)
		if opts.ScopePath != "" {
			fmt.Printf(" (%s)", opts.ScopePath)
		}
		fmt.Println()
		if len(opts.Tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(opts.Tags, ", "))
		}
		fmt.Printf("  Content: %s\n", util.TruncateLine(opts.Content, 60))
	} else {
		// Proposal output
		fmt.Printf("📥 Proposal created (%s): %s\n", kind, id)
		fmt.Printf("  Scope: %s", opts.Scope)
		if opts.ScopePath != "" {
			fmt.Printf(" (%s)", opts.ScopePath)
		}
		fmt.Println()
		fmt.Printf("  Classification: %.0f%% confidence\n", classification.Confidence*100)
		fmt.Printf("  Content: %s\n", util.TruncateLine(opts.Content, 60))
		fmt.Println("  Use 'palace proposals' to review and approve.")
	}
	return nil
}
