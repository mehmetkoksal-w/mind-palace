package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: extract_changelog <version>")
		os.Exit(2)
	}

	version := strings.TrimSpace(os.Args[1])
	version = strings.TrimPrefix(version, "v")
	if version == "" {
		fmt.Fprintln(os.Stderr, "version is required")
		os.Exit(2)
	}

	body, err := extractChangelog("CHANGELOG.md", version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Print(body)
}

func extractChangelog(path, version string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

	headerPrefix := "## [" + version + "]"
	scanner := bufio.NewScanner(file)
	inSection := false
	var lines []string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## [") {
			if inSection {
				break
			}
			if strings.HasPrefix(line, headerPrefix) {
				inSection = true
			}
			continue
		}

		if inSection {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan %s: %w", path, err)
	}
	if !inSection {
		return "", fmt.Errorf("version %s not found in %s", version, path)
	}

	lines = trimEmptyLines(lines)
	return strings.Join(lines, "\n"), nil
}

func trimEmptyLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[start:end]
}
