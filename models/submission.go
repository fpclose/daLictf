// file: models/submission.go
package models

import (
	"time"
)

type Submission struct {
	ID          uint64 `gorm:"primarykey"`
	ChallengeID uint32 `gorm:"not null"`
	UserID      uint32 `gorm:"not null"`
	TeamID      uint32 `gorm:"not null"`
	IsCorrect   bool
	SubmittedAt time.Time
}

func (Submission) TableName() string {
	return "dalictf_submission"
}
