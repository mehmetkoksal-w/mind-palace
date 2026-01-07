# Performance Benchmarks

**Status:** Initialized on January 7, 2026

**Test Environment:**

- **Hardware:** Intel Core i7-13700K, 32GB RAM
- **OS:** Windows 11 Pro (x64)
- **Go Version:** 1.25.0
- **Date:** 2026-01-07

## Benchmark Execution

Benchmarks are defined in `benchmarks_test.go` and can be executed with:

```bash
go test -bench=. -benchmem -run=^# ./tests
```

## Results

**Current Status:** Pending execution due to CGO configuration on Windows.

### Methodology

- **Indexing:** Measure scan and collect operations across varying file counts (100/1k files).
- **WriteScan:** Track write operation performance.
- **LLM Requests:** Time requests to configured LLM backends (Ollama/OpenAI).

### Known Bottlenecks & Recommendations

- CGO dependencies (tree-sitter) require MinGW on Windows; CI (Linux) will provide canonical metrics.
- Package structure in `apps/cli/tests` requires consolidation (currently mixed packages: tests, integration).
- Core unit tests validated: config, corridor, signal, project, LLM (90.6% coverage).
