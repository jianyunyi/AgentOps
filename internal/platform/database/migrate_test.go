package database

import "testing"

func TestMigrationVersionsAreOrdered(t *testing.T) {
	versions := MigrationVersions()
	if len(versions) < 2 || versions[0] != 1 || versions[1] != 2 {
		t.Fatalf("unexpected migration versions: %v", versions)
	}
}
