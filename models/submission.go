// file: models/submission.go
package models

import (
	"time"
)

// Submission 结构体现在对应解题得分表 dalictf_problem_solving_record
type Submission struct {
	ID          uint32    `gorm:"primarykey" json:"id"`
	ChallengeID uint32    `gorm:"uniqueIndex:unique_team_challenge;not null" json:"challenge_id"`
	TeamID      uint32    `gorm:"uniqueIndex:unique_team_challenge;not null" json:"team_id"`
	UserID      uint32    `gorm:"not null" json:"user_id"`
	Score       uint      `gorm:"not null" json:"score"`
	SolvingTime time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"solving_time"`
}

func (Submission) TableName() string {
	return "dalictf_problem_solving_record"
}
