package audit

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeRepository struct{ records []Record }

func (f *fakeRepository) List(_ context.Context, tenantID string, offset, limit int, action string) ([]Record, int64, error) {
	var result []Record
	for _, record := range f.records {
		if record.TenantID == tenantID && (action == "" || record.Action == action) {
			result = append(result, record)
		}
	}
	if offset >= len(result) {
		return []Record{}, int64(len(result)), nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], int64(len(result)), nil
}

func (f *fakeRepository) Append(_ context.Context, record *Record) error {
	f.records = append(f.records, *record)
	return nil
}

func TestAppendRedactsSecretsFromSnapshots(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	err := svc.Record(context.Background(), RecordInput{
		TenantID: "ten_001", ActorID: "usr_001", Action: "agent.key.rotate", ResourceType: "agent", ResourceID: "agt_001",
		Before: map[string]any{"api_key": "ag_live_secret", "password": "super-secret"},
		After:  map[string]any{"key_hash": "hash", "api_key": "ag_live_new"}, RequestID: "req_001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.records) != 1 {
		t.Fatal("one audit record expected")
	}
	var before, after map[string]any
	if err := json.Unmarshal(repo.records[0].Before, &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(repo.records[0].After, &after); err != nil {
		t.Fatal(err)
	}
	if _, ok := before["api_key"]; ok {
		t.Fatal("raw API key must not be stored")
	}
	if _, ok := before["password"]; ok {
		t.Fatal("password must not be stored")
	}
	if _, ok := after["api_key"]; ok {
		t.Fatal("raw API key must not be stored")
	}
}

func TestAuditRecordIsTenantScoped(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	if err := svc.Record(context.Background(), RecordInput{TenantID: "ten_001", ActorID: "usr_001", Action: "agent.create", ResourceType: "agent", ResourceID: "agt_001"}); err != nil {
		t.Fatal(err)
	}
	if repo.records[0].TenantID != "ten_001" {
		t.Fatal("audit record must preserve tenant scope")
	}
}
