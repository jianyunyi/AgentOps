package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMigrationStatusUnavailable = errors.New("agent credential migration status unavailable")
	ErrInvalidMigrationQuery      = errors.New("invalid agent credential migration query")
	ErrLegacyCredentialsRemain    = errors.New("legacy agent credentials remain")
)

const (
	defaultMigrationPageSize = 20
	maxMigrationPageSize     = 100
)

type CredentialMigrationSummary struct {
	TotalAgents    int64 `json:"total_agents"`
	MigratedAgents int64 `json:"migrated_agents"`
	LegacyAgents   int64 `json:"legacy_agents"`
}

type LegacyAgent struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Environment string     `json:"environment"`
	Status      string     `json:"status"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

type CredentialMigrationPagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type CredentialMigrationStatus struct {
	Summary    CredentialMigrationSummary    `json:"summary"`
	Agents     []LegacyAgent                 `json:"agents"`
	Pagination CredentialMigrationPagination `json:"pagination"`
}

type CredentialMigrationRepository interface {
	GetCredentialMigrationSummary(ctx context.Context, tenantID string) (CredentialMigrationSummary, error)
	ListLegacyAgents(ctx context.Context, tenantID, query string, page, pageSize int) ([]LegacyAgent, int64, error)
}

type LegacyCredentialCounter interface {
	CountLegacyCredentials(ctx context.Context) (int64, error)
}

func (s *Service) GetCredentialMigrationStatus(ctx context.Context, tenantID, query string, page, pageSize int) (CredentialMigrationStatus, error) {
	if strings.TrimSpace(tenantID) == "" || page < 1 || pageSize < 1 || pageSize > maxMigrationPageSize {
		return CredentialMigrationStatus{}, ErrInvalidMigrationQuery
	}
	migrationRepo, ok := s.repo.(CredentialMigrationRepository)
	if !ok {
		return CredentialMigrationStatus{}, ErrMigrationStatusUnavailable
	}
	query = strings.TrimSpace(query)
	if len(query) > 128 {
		return CredentialMigrationStatus{}, ErrInvalidMigrationQuery
	}
	summary, err := migrationRepo.GetCredentialMigrationSummary(ctx, tenantID)
	if err != nil {
		return CredentialMigrationStatus{}, fmt.Errorf("get migration summary: %w", err)
	}
	agents, total, err := migrationRepo.ListLegacyAgents(ctx, tenantID, query, page, pageSize)
	if err != nil {
		return CredentialMigrationStatus{}, fmt.Errorf("list legacy agents: %w", err)
	}
	return CredentialMigrationStatus{
		Summary:    summary,
		Agents:     agents,
		Pagination: CredentialMigrationPagination{Page: page, PageSize: pageSize, Total: total},
	}, nil
}

func CheckSignatureMigrationReady(ctx context.Context, counter LegacyCredentialCounter) error {
	legacyCount, err := counter.CountLegacyCredentials(ctx)
	if err != nil {
		return fmt.Errorf("check legacy agent credentials: %w", err)
	}
	if legacyCount > 0 {
		return fmt.Errorf("%d active agents still use legacy credentials: %w", legacyCount, ErrLegacyCredentialsRemain)
	}
	return nil
}
