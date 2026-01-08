package integration_test

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
)

func generateTestFiles(count int) (string, error) {
	tmpDir, err := os.MkdirTemp("", "mp-bench-*")
	if err != nil {
		return "", err
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	langs := []string{"go", "ts", "py"}

	for i := 0; i < count; i++ {
		lang := langs[i%len(langs)]
		sub := fmt.Sprintf("dir%d/dir%d", r.Intn(5), r.Intn(5))
		dir := filepath.Join(tmpDir, sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}

		var filename string
		var content []byte
		switch lang {
		case "go":
			filename = filepath.Join(dir, fmt.Sprintf("file%d.go", i))
			content = []byte(fmt.Sprintf("package bench\n\nimport \"fmt\"\n\n// File %d\nfunc Fn%d(x int) int {\n\tfmt.Println(x)\n\treturn x*x\n}\n\ntype T%d struct { A int }\n", i, i, i))
		case "ts":
			filename = filepath.Join(dir, fmt.Sprintf("file%d.ts", i))
			content = []byte(fmt.Sprintf("export function fn%d(x:number){ return x*x }\nexport class C%d { a:number=0 }\n", i, i))
		case "py":
			filename = filepath.Join(dir, fmt.Sprintf("file%d.py", i))
			content = []byte(fmt.Sprintf("def fn%d(x):\n\treturn x*x\n\nclass C%d:\n\tdef __init__(self):\n\t\tself.a = 0\n", i, i))
		}
		if err := os.WriteFile(filename, content, 0o644); err != nil {
			return "", err
		}
	}

	return tmpDir, nil
}

func cleanupTestFiles(dir string) {
	_ = os.RemoveAll(dir)
}

func BenchmarkBuildFileRecords100(b *testing.B) {
	root, err := generateTestFiles(100)
	if err != nil {
		b.Fatal(err)
	}
	defer cleanupTestFiles(root)

	guard := config.Guardrails{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := index.BuildFileRecords(root, guard)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildFileRecords1k(b *testing.B) {
	root, err := generateTestFiles(1000)
	if err != nil {
		b.Fatal(err)
	}
	defer cleanupTestFiles(root)

	guard := config.Guardrails{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := index.BuildFileRecords(root, guard)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteScan100(b *testing.B) {
	root, err := generateTestFiles(100)
	if err != nil {
		b.Fatal(err)
	}
	defer cleanupTestFiles(root)

	records, err := index.BuildFileRecords(root, config.Guardrails{})
	if err != nil {
		b.Fatal(err)
	}

	dbPath := filepath.Join(root, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := index.WriteScan(db, root, records, time.Now())
		if err != nil {
			b.Fatal(err)
		}
	}
}
