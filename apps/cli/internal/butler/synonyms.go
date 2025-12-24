package butler

import "strings"

// ProgrammingSynonyms maps common programming terms to their synonyms.
// This enables semantic search where "get" also finds "fetch", "retrieve", etc.
var ProgrammingSynonyms = map[string][]string{
	// CRUD operations
	"get":    {"fetch", "retrieve", "read", "load", "obtain", "find"},
	"fetch":  {"get", "retrieve", "load"},
	"set":    {"put", "write", "store", "save", "assign", "update"},
	"create": {"new", "make", "init", "build", "generate", "add", "insert"},
	"delete": {"remove", "destroy", "drop", "clear", "erase", "del"},
	"update": {"modify", "change", "edit", "patch", "alter", "set"},

	// Control flow
	"handle":  {"process", "manage", "deal"},
	"handler": {"controller", "processor", "listener", "callback"},
	"run":     {"execute", "start", "launch", "invoke", "call"},
	"stop":    {"halt", "terminate", "end", "kill", "abort"},

	// Data structures
	"list":   {"array", "slice", "collection", "items"},
	"map":    {"dict", "dictionary", "hash", "object", "table"},
	"queue":  {"buffer", "channel"},
	"config": {"configuration", "settings", "options", "preferences", "conf", "cfg"},

	// Error handling
	"error":     {"err", "exception", "fault", "failure"},
	"err":       {"error", "exception"},
	"exception": {"error", "err"},
	"validate":  {"check", "verify", "ensure", "assert"},
	"check":     {"validate", "verify", "test"},

	// Common patterns
	"parse":     {"decode", "deserialize", "unmarshal", "read"},
	"serialize": {"encode", "marshal", "stringify", "dump"},
	"encode":    {"serialize", "marshal"},
	"decode":    {"parse", "deserialize", "unmarshal"},
	"convert":   {"transform", "cast", "translate", "map"},
	"format":    {"render", "display", "stringify"},

	// Authentication/Authorization
	"auth":         {"authentication", "authorization", "login"},
	"authenticate": {"login", "signin", "auth"},
	"authorize":    {"permit", "allow", "grant"},
	"login":        {"signin", "authenticate", "logon"},
	"logout":       {"signout", "logoff"},
	"user":         {"account", "profile", "member"},
	"token":        {"key", "credential", "jwt"},

	// HTTP/API
	"request":  {"req", "call", "query"},
	"response": {"res", "reply", "result"},
	"send":     {"dispatch", "emit", "transmit", "post"},
	"receive":  {"accept", "get", "consume"},
	"api":      {"endpoint", "service", "interface"},

	// Database
	"query":      {"select", "find", "search", "fetch"},
	"insert":     {"add", "create", "put"},
	"database":   {"db", "store", "repository"},
	"repository": {"repo", "store", "database"},
	"cache":      {"store", "buffer", "memo"},

	// File operations
	"file":   {"document", "doc"},
	"read":   {"load", "get", "fetch", "open"},
	"write":  {"save", "store", "put"},
	"path":   {"filepath", "route", "location"},
	"open":   {"read", "load", "access"},
	"close":  {"shutdown", "dispose", "cleanup"},
	"save":   {"write", "store", "persist"},

	// Async operations
	"async":   {"asynchronous", "concurrent", "parallel"},
	"await":   {"wait", "block"},
	"promise": {"future", "task", "async"},
	"channel": {"chan", "pipe", "stream"},

	// Testing
	"test":   {"spec", "check", "verify"},
	"mock":   {"stub", "fake", "spy"},
	"assert": {"expect", "verify", "check"},

	// Logging
	"log":   {"logger", "logging", "trace"},
	"debug": {"trace", "verbose"},
	"info":  {"information", "message"},
	"warn":  {"warning", "alert"},

	// Misc programming
	"util":     {"utility", "helper", "common"},
	"helper":   {"util", "utility", "tool"},
	"function": {"func", "fn", "method", "procedure"},
	"method":   {"func", "function", "member"},
	"class":    {"type", "struct", "model"},
	"interface": {"contract", "protocol", "trait"},
	"module":   {"package", "lib", "library"},
	"import":   {"include", "require", "use"},
	"export":   {"expose", "public"},
	"private":  {"internal", "hidden"},
	"public":   {"exported", "exposed"},
	"init":     {"initialize", "setup", "bootstrap", "start"},
	"cleanup":  {"teardown", "dispose", "destroy"},
}

// expandWithSynonyms takes a list of tokens and adds synonyms
func expandWithSynonyms(tokens []string) []string {
	result := make([]string, 0, len(tokens)*2)
	seen := make(map[string]bool)

	for _, token := range tokens {
		lower := strings.ToLower(token)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, token)
		}

		// Add synonyms for this token
		if synonyms, ok := ProgrammingSynonyms[lower]; ok {
			for _, syn := range synonyms {
				if !seen[syn] {
					seen[syn] = true
					result = append(result, syn)
				}
			}
		}
	}

	return result
}

// GetSynonyms returns synonyms for a given term
func GetSynonyms(term string) []string {
	lower := strings.ToLower(term)
	if synonyms, ok := ProgrammingSynonyms[lower]; ok {
		return synonyms
	}
	return nil
}
