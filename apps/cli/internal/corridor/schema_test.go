package corridor

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCorridorSchemaVersion(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := initCorridorDB(db); err != nil {
		t.Fatalf("initCorridorDB() error = %v", err)
	}

	version, err := GetCorridorSchemaVersion(db)
	if err != nil {
		t.Fatalf("GetCorridorSchemaVersion() error = %v", err)
	}
	if version != 0 {
		t.Fatalf("schema version = %d, want 0", version)
	}
}
