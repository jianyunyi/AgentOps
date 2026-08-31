package tenant

import "time"

type Tenant struct {
	ID        string `gorm:"primaryKey;size:32"`
	Name      string `gorm:"size:128;not null"`
	Status    string `gorm:"size:16;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
