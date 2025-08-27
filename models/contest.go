// file: models/contest.go
package models

import (
	"time"
)

// ContestStatus 定义比赛状态
type ContestStatus string

const (
	ContestStatusPreparing ContestStatus = "preparing"
	ContestStatusRunning   ContestStatus = "running"
	ContestStatusEnded     ContestStatus = "ended"
)

// Contest 对应 dalictf_contest 表 (已添加 JSON 绑定标签)
type Contest struct {
	ID           uint          `gorm:"primarykey" json:"id,omitempty"`
	ContestName  string        `gorm:"size:100;not null" json:"contest_name"`
	CoverImage   string        `gorm:"size:255" json:"cover_image"`
	Description  string        `gorm:"type:text" json:"description"`
	StartTime    time.Time     `gorm:"not null" json:"start_time" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	EndTime      time.Time     `gorm:"not null" json:"end_time" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	OrganizerURL string        `gorm:"size:255" json:"organizer_url"`
	Status       ContestStatus `gorm:"type:enum('preparing','running','ended');default:'preparing'" json:"status,omitempty"`
	CreatedAt    time.Time     `json:"created_at,omitempty"`
	UpdatedAt    time.Time     `json:"updated_at,omitempty"`
}

func (Contest) TableName() string {
	return "dalictf_contest"
}
