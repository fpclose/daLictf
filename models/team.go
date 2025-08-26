// file: models/team.go
package models

import (
	"time"
)

// 自定义队伍状态类型
type TeamStatus string

const (
	TeamStatusActive TeamStatus = "active"
	TeamStatusBanned TeamStatus = "banned"
	TeamStatusHidden TeamStatus = "hidden"
)

type Team struct {
	ID             uint32       `gorm:"primarykey" json:"id"`
	TeamName       string       `gorm:"size:100;unique;not null" json:"team_name"`
	LeaderID       uint32       `gorm:"not null" json:"leader_id"`
	Leader         User         `gorm:"foreignKey:LeaderID" json:"leader"`
	SchoolID       *uint32      `json:"school_id"`
	School         *School      `gorm:"foreignKey:SchoolID" json:"school"`
	Track          UserTrack    `gorm:"type:enum('freshman','advanced','society');not null" json:"track"`
	InvitationCode string       `gorm:"size:20;unique;not null" json:"invitation_code"`
	TeamDescribe   string       `gorm:"type:text" json:"team_describe"`
	TeamStatus     TeamStatus   `gorm:"type:enum('active','banned','hidden');default:'active'" json:"team_status"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	Members        []TeamMember `gorm:"foreignKey:TeamID" json:"members"`
}

func (Team) TableName() string {
	return "dalictf_team"
}
