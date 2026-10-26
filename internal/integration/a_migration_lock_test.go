package integration_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"agentscope/internal/auth"
	"agentscope/internal/platform/database"
	"gorm.io/gorm"
)

func TestConcurrentDatabaseMigrationIsSerialized(t *testing.T) {
	if os.Getenv("AGENTSCOPE_INTEGRATION") != "1" {
		t.Skip("set AGENTSCOPE_INTEGRATION=1 to run MySQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db, err := database.OpenWithPool(ctx, os.Getenv("MYSQL_DSN"), 4, 2, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- database.Migrate(db)
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent migration failed: %v", err)
		}
	}

	var applied int64
	if err := db.Table("schema_migrations").Count(&applied).Error; err != nil {
		t.Fatal(err)
	}
	if want := int64(len(database.MigrationVersions())); applied != want {
		t.Fatalf("applied migrations = %d, want %d", applied, want)
	}

	stamp := time.Now().UnixNano()
	legacyID := fmt.Sprintf("ses_legacy_%d", stamp)
	hashedID := auth.SessionLookupKey("migration-test-secret", fmt.Sprintf("ses_hashed_%d", stamp))
	for _, id := range []string{legacyID, hashedID} {
		if err := db.Create(&auth.Session{ID: id, UserID: fmt.Sprintf("user_%d", stamp), TenantID: fmt.Sprintf("tenant_%d", stamp), Role: auth.RoleOwner, ExpiresAt: time.Now().Add(time.Hour)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := database.MigrateWithOptions(db, database.MigrateOptions{PurgeLegacySessions: true}); err != nil {
		t.Fatal(err)
	}
	var stored auth.Session
	if err := db.Where("id = ?", legacyID).First(&stored).Error; err != gorm.ErrRecordNotFound {
		t.Fatalf("legacy plaintext session should be purged, err = %v", err)
	}
	if err := db.Where("id = ?", hashedID).First(&stored).Error; err != nil {
		t.Fatalf("HMAC session should survive legacy cleanup: %v", err)
	}
}
