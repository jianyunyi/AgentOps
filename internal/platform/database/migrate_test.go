package database

import "testing"

func TestMigrationVersionsAreOrdered(t *testing.T) {
	versions := MigrationVersions()
	if len(versions) < 3 || versions[0] != 1 || versions[1] != 2 || versions[2] != 3 {
		t.Fatalf("unexpected migration versions: %v", versions)
	}
}
