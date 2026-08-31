package database

import (
	"agentscope/internal/agent"
	"agentscope/internal/audit"
	"agentscope/internal/outbox"
	"agentscope/internal/risk"
	"agentscope/internal/auth"
	"agentscope/internal/tenant"
	"agentscope/internal/trace"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&agent.Agent{},
		&agent.AgentCredential{},
		&auth.User{},
		&auth.Session{},
		&tenant.Tenant{},
		&audit.Record{},
		&outbox.Event{},
		&trace.Trace{},
		&trace.Span{},
		&trace.EventRecord{},
		&risk.RiskEvent{},
	)
}
