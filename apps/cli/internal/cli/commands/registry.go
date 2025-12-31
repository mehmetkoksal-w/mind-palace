// Package commands provides the command implementations for the CLI.
package commands

// CommandFunc is the function signature for CLI commands.
type CommandFunc func(args []string) error

// Command represents a CLI command with its metadata.
type Command struct {
	// Name is the primary name of the command
	Name string
	// Aliases are alternative names for the command
	Aliases []string
	// Description is a brief description shown in help
	Description string
	// Run is the function that executes the command
	Run CommandFunc
}

// registry holds all registered commands
var registry = make(map[string]*Command)

// Register adds a command to the registry.
// The command is registered under its name and all aliases.
func Register(cmd *Command) {
	registry[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		registry[alias] = cmd
	}
}

// Get retrieves a command by name or alias.
func Get(name string) (*Command, bool) {
	cmd, ok := registry[name]
	return cmd, ok
}

// List returns all unique commands (without alias duplicates).
func List() []*Command {
	seen := make(map[string]bool)
	var commands []*Command
	for _, cmd := range registry {
		if !seen[cmd.Name] {
			seen[cmd.Name] = true
			commands = append(commands, cmd)
		}
	}
	return commands
}
