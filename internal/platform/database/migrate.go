package database

import (
	"agentscope/internal/agent"
	"agentscope/internal/auth"
	"agentscope/internal/trace"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&agent.Agent{},
		&agent.AgentCredential{},
		&auth.User{},
		&auth.Session{},
		&trace.Trace{},
		&trace.Span{},
		&trace.EventRecord{},
	)
}
