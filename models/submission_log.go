// file: models/submission_log.go
package models

import (
	"time"
)

type FlagResult string

const (
	FlagResultCorrect   FlagResult = "correct"
	FlagResultWrong     FlagResult = "wrong"
	FlagResultDuplicate FlagResult = "duplicate"
)

// SubmissionLog 对应 dalictf_flag_information 表
type SubmissionLog struct {
	ID             uint64     `gorm:"primarykey"`
	ChallengeID    uint32     `gorm:"not null"`
	TeamID         uint32     `gorm:"not null"`
	UserID         uint32     `gorm:"not null"`
	SubmittedFlag  string     `gorm:"size:255;not null"`
	FlagResult     FlagResult `gorm:"type:enum('correct','wrong','duplicate');not null"`
	SubmissionTime time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	IPAddress      string     `gorm:"size:45"`
	Suspected      bool       `gorm:"default:0"` // 新增字段
}

func (SubmissionLog) TableName() string {
	return "dalictf_flag_information"
}
