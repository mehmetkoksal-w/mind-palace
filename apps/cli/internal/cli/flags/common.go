package flags

import "flag"

// CommonFlags holds pointers to commonly used flag values.
// This allows commands to share flag definitions consistently.
type CommonFlags struct {
	Root    *string
	Limit   *int
	Verbose *bool
	Query   *string
}

// AddRootFlag adds --root and -r flags for workspace root.
func AddRootFlag(fs *flag.FlagSet) *string {
	root := fs.String("root", ".", "workspace root")
	fs.StringVar(root, "r", ".", "workspace root (shorthand)")
	return root
}

// AddLimitFlag adds --limit and -l flags for result limits.
func AddLimitFlag(fs *flag.FlagSet, defaultValue int) *int {
	limit := fs.Int("limit", defaultValue, "maximum results")
	fs.IntVar(limit, "l", defaultValue, "maximum results (shorthand)")
	return limit
}

// AddVerboseFlag adds --verbose and -v flags for verbose output.
func AddVerboseFlag(fs *flag.FlagSet) *bool {
	verbose := fs.Bool("verbose", false, "show detailed output")
	fs.BoolVar(verbose, "v", false, "show detailed output (shorthand)")
	return verbose
}

// AddForceFlag adds --force and -f flags for overwrite operations.
func AddForceFlag(fs *flag.FlagSet) *bool {
	force := fs.Bool("force", false, "overwrite existing files")
	fs.BoolVar(force, "f", false, "overwrite existing files (shorthand)")
	return force
}

// AddQuietFlag adds --quiet and -q flags for quiet mode.
func AddQuietFlag(fs *flag.FlagSet) *bool {
	quiet := fs.Bool("quiet", false, "suppress non-essential output")
	fs.BoolVar(quiet, "q", false, "suppress non-essential output (shorthand)")
	return quiet
}

// AddPortFlag adds --port and -p flags for server port.
func AddPortFlag(fs *flag.FlagSet, defaultPort int) *int {
	port := fs.Int("port", defaultPort, "server port")
	fs.IntVar(port, "p", defaultPort, "server port (shorthand)")
	return port
}
