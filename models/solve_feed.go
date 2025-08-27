// file: models/solve_feed.go
package models

import (
	"time"
)

// SolveFeed 对应 dalictf_solve_feed 缓存表
type SolveFeed struct {
	ID            uint64    `gorm:"primarykey"`
	ChallengeID   uint32    `gorm:"not null"`
	ChallengeName string    `gorm:"size:100;not null"`
	TeamID        uint32    `gorm:"not null"`
	TeamName      string    `gorm:"size:100;not null"`
	SchoolName    *string   `gorm:"size:100"`
	Score         uint      `gorm:"not null"`
	SolvingTime   time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

func (SolveFeed) TableName() string {
	return "dalictf_solve_feed"
}
