// file: models/scoreboard.go
package models

import (
	"time"
)

// ScoreboardTrack 定义了排行榜的赛道分类
type ScoreboardTrack string

const (
	// TrackFreshman, TrackAdvanced, TrackSociety 已在 user.go 中定义，此处不再重复
	// 这里我们只定义新增的总榜常量
	TrackOverall ScoreboardTrack = "overall"
)

// Scoreboard 对应 dalictf_scoreboard 缓存表
type Scoreboard struct {
	ID            uint            `gorm:"primarykey"`
	TeamID        uint32          `gorm:"not null"`
	TeamName      string          `gorm:"size:100;not null"`
	SchoolName    *string         `gorm:"size:100"`
	Track         ScoreboardTrack `gorm:"type:enum('freshman','advanced','society','overall');not null"`
	Score         uint            `gorm:"not null"`
	LastSolveTime *time.Time
	Rank          uint `gorm:"not null"`
	UpdatedAt     time.Time
}

func (Scoreboard) TableName() string {
	return "dalictf_scoreboard"
}
