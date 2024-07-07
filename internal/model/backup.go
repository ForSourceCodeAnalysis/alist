package model

import (
	"time"
)

type Backup struct {
	ID           uint      `json:"id" gorm:"primaryKey"` // unique key
	FilePath     string    `json:"file_path" gorm:"unique" binding:"required"`
	LastModified time.Time `json:"last_modified"`
	Tag          string    `json:"tag"`
}
