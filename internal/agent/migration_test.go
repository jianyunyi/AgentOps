package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type migrationRepository struct {
	fakeRepository
	summary       CredentialMigrationSummary
	agents        []LegacyAgent
	total         int64
	err           error
	count         int64
	countErr      error
	requestedPage int
	requestedSize int
	requestedQ    string
}

func (r *migrationRepository) GetCredentialMigrationSummary(context.Context, string) (CredentialMigrationSummary, error) {
	return r.summary, r.err
}

func (r *migrationRepository) ListLegacyAgents(_ context.Context, _ string, q string, page, pageSize int) ([]LegacyAgent, int64, error) {
	r.requestedPage, r.requestedSize, r.requestedQ = page, pageSize, q
	return r.agents, r.total, r.err
}

func (r *migrationRepository) CountLegacyCredentials(context.Context) (int64, error) {
	return r.count, r.countErr
}

func TestGetCredentialMigrationStatusValidatesAndScopesQuery(t *testing.T) {
	lastUsedAt := time.Now().UTC()
	repo := &migrationRepository{
		summary: CredentialMigrationSummary{TotalAgents: 3, MigratedAgents: 2, LegacyAgents: 1},
		agents:  []LegacyAgent{{ID: "agt_legacy", Name: "legacy", LastUsedAt: &lastUsedAt}},
		total:   1,
	}
	svc := NewService(repo)

	status, err := svc.GetCredentialMigrationStatus(context.Background(), "tenant_001", "legacy", 2, 25)
	if err != nil {
		t.Fatalf("GetCredentialMigrationStatus() error = %v", err)
	}
	if status.Summary.LegacyAgents != 1 || len(status.Agents) != 1 || status.Pagination.Total != 1 {
		t.Fatalf("unexpected migration status: %+v", status)
	}
	if repo.requestedPage != 2 || repo.requestedSize != 25 || repo.requestedQ != "legacy" {
		t.Fatalf("unexpected repository arguments: page=%d size=%d q=%q", repo.requestedPage, repo.requestedSize, repo.requestedQ)
	}
}

func TestGetCredentialMigrationStatusRejectsInvalidPagination(t *testing.T) {
	svc := NewService(&migrationRepository{})
	if _, err := svc.GetCredentialMigrationStatus(context.Background(), "tenant_001", "", 0, 20); !errors.Is(err, ErrInvalidMigrationQuery) {
		t.Fatalf("page 0 error = %v, want ErrInvalidMigrationQuery", err)
	}
	if _, err := svc.GetCredentialMigrationStatus(context.Background(), "tenant_001", "", 1, 101); !errors.Is(err, ErrInvalidMigrationQuery) {
		t.Fatalf("page size 101 error = %v, want ErrInvalidMigrationQuery", err)
	}
}

func TestSignatureMigrationPreflightRejectsLegacyCredentials(t *testing.T) {
	err := CheckSignatureMigrationReady(context.Background(), &migrationRepository{count: 2})
	if err == nil || !errors.Is(err, ErrLegacyCredentialsRemain) {
		t.Fatalf("preflight error = %v, want ErrLegacyCredentialsRemain", err)
	}
	if !errors.Is(err, ErrLegacyCredentialsRemain) || !strings.Contains(err.Error(), "2 active agents still use legacy credentials") {
		t.Fatalf("unexpected preflight error: %v", err)
	}
}

func TestSignatureMigrationPreflightPropagatesDatabaseError(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	err := CheckSignatureMigrationReady(context.Background(), &migrationRepository{countErr: databaseErr})
	if !errors.Is(err, databaseErr) {
		t.Fatalf("preflight error = %v, want database error", err)
	}
}
