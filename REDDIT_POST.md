# Reddit Post: Mind Palace Context System

## Before Recording

### Check Installation
```bash
asciinema --version
```

### Recommended Settings
- Terminal size: 80x24 or 100x30
- Font: Monospace, 14pt or larger
- Theme: Light background (most readable)

---

## Recording Instructions

### Step 1: Prepare the Project
Open terminal in your mind-palace directory:
```bash
cd /Users/mehmetkoksal/Documents/Projects/Personal/mind-palace
```

### Step 2: Create a Demo Project
Create a small temporary project to demonstrate indexing:
```bash
mkdir -f /tmp/demo-project
cd /tmp/demo-project

# Create sample files
mkdir -p auth api utils

# auth/login.go
cat > auth/login.go << 'EOF'
package auth

func ValidatePassword(password string) bool {
    if len(password) < 8 {
        return false
    }
    return true
}

func HashPassword(password string) string {
    // Using bcrypt for secure hashing
    return "hashed_value"
}
EOF

# auth/middleware.go
cat > auth/middleware.go << 'EOF'
package auth

import "net/http"

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "missing token", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
EOF

# api/server.go
cat > api/server.go << 'EOF'
package api

import (
    "net/http"
    "demo-project/auth"
)

func StartServer() {
    http.HandleFunc("/login", auth.ValidatePassword)
}
EOF
```

### Step 3: Clear Previous Demo State (if any)
```bash
rm -rf /tmp/demo-project/.palace
```

### Step 4: Start Recording
```bash
asciinema rec /tmp/mind-palace-demo.cast
```

### Step 5: Execute This Exact Sequence
Type each command exactly as shown. Pause briefly between commands for readability.

```bash
# 1. Initialize
./palace init
# Select "Go" as language, accept defaults
# Name: demo-project

# 2. Scan the project
./palace scan
# Watch it index 3 files

# 3. Explore authentication
./palace explore "where is authentication handled"
# Wait for results

# 4. Show more specific query
./palace explore "password validation"
# Wait for results

# 5. List rooms
./palace rooms

# 6. Brief a file
./palace brief auth/login.go

# 7. Exit
./palace help
```

### Step 6: Stop Recording
Press `Ctrl+C` then `exit` to finish recording.

### Step 7: Generate GIF
```bash
# Install svgterm if needed (optional, for better quality)
# svg2gif preferred for crisp rendering

# Convert to GIF using agg (asciinema gif)
agg /tmp/mind-palace-demo.cast mind-palace-demo.gif

# Alternative: upload .cast file directly to asciinema.org
# Reddit supports direct .cast embedding via asciinema.org links
```

---

## Recording Tips

**Timing**: Total recording should be 20-35 seconds.

**Common Mistakes**:
- Typing too fast (viewers can't follow)
- Long pauses (recording feels dead)
- Scrolling while text is output (loses context)

**What to Avoid**:
- Don't show the full scan output (too long)
- Don't demo governanceposals (too complex for quick/pro demo)
- Don't open the dashboard (requires browser switch)

**If Recording Fails**:
- Start over: `asciinema rec /tmp/demo.cast`
- You can re-record multiple times

---

## Reddit Post

---

**Title**: I built a context system for codebases that doesn't use embeddings

**Body**:

I've been working on a problem that AI agents face: every session is a fresh start. No memory of what was decided, why certain code exists, or what patterns emerged from previous work.

Most solutions use embeddings - vector databases that store "semantic meaning." They work until they don't. You query "auth" and get 15 files that mention auth in some capacity, none of which are the actual authentication logic. The AI guesses. You verify manually. Context is still lost.

This project takes a different approach. It builds a deterministic index of your codebase using Tree-sitter for structural parsing and SQLite FTS5 for exact matching. No embeddings. No guesswork. When you ask "where is authentication handled," it returns file paths, line numbers, and the actual code structure that handles auth.

Key features:

- Session tracking: agents log their activities, decisions, and learnings
- Corridors: share knowledge across multiple repositories
- Governance: proposal/approval workflow for knowledge authority
- MCP server: works with Claude Desktop, Cursor, and other MCP-enabled agents
- VS Code extension: sidebar, HUD, and auto-sync

The core idea is from the Method of Loci - a memory technique used since ancient Greece. Instead of placing memories in physical locations, you place knowledge about code in a structured palace.

It's alpha software (v0.3.1) and needs more testing, but I'm posting now because the approach seems genuinely different from what's out there.

GitHub: https://github.com/koksalmehmet/mind-palace

Demo: https://asciinema.org/a/XXXXXX (replace with your recording)

Questions welcome.

---

## After Recording

1. Upload .cast to asciinema.org
2. Replace `XXXXXX` in the post with your asciinema ID
3. Review the post for tone - remove any phrasing that sounds like marketing
4. Post to r/programming or r/agnoster
5. Check comments for legitimate questions

---

## Alternative: Text-Only Post

If you skip the recording, replace the demo link with this:

```
Example query output:

$ palace explore "password validation"
Found 2 matches:

1. auth/login.go:5-9
   Function: ValidatePassword
   Checks password length >= 8 characters

2. auth/login.go:12-14
   Function: HashPassword
   Notes: Using bcrypt for secure hashing
```

This shows the deterministic output without needing video.

---

## Notes

- The post intentionally avoids words like "revolutionary," "game-changing," "innovative," "cutting-edge"
- It acknowledges alpha status upfront
- It describes concrete technical differences (Tree-sitter, SQLite FTS5)
- It provides a link, not a hard sell
- The demo is opt-in, not required to understand the concept
