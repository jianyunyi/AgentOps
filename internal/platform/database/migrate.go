package database

import (
	"agentscope/internal/agent"
	"agentscope/internal/trace"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&agent.Agent{},
		&agent.AgentCredential{},
		&trace.Trace{},
		&trace.Span{},
		&trace.EventRecord{},
	)
}
