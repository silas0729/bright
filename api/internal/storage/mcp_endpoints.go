package storage

import "time"

type LearnerMCPEndpoint struct {
	ID                uint   `gorm:"primaryKey"`
	LearnerUserID     uint   `gorm:"not null;index"`
	Name              string `gorm:"size:120;not null"`
	URL               string `gorm:"size:1000;not null"`
	Description       string `gorm:"size:500"`
	Enabled           bool   `gorm:"not null;default:true;index"`
	TokenQueryParam   string `gorm:"size:80"`
	SubjectQueryParam string `gorm:"size:80"`
	ConnectionStatus  string `gorm:"size:32;not null;default:disconnected;index"`
	LastError         string `gorm:"size:1000"`
	ConnectedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
