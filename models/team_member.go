// file: models/team_member.go
package models

import "time"

// 自定义队伍角色类型
type TeamMemberRole string

const (
	TeamRoleLeader TeamMemberRole = "leader"
	TeamRoleMember TeamMemberRole = "member"
)

type TeamMember struct {
	ID       uint32         `gorm:"primarykey"`
	TeamID   uint32         `gorm:"uniqueIndex:unique_team_user;not null"`
	UserID   uint32         `gorm:"uniqueIndex:unique_team_user;not null"`
	User     User           `gorm:"foreignKey:UserID"`
	Role     TeamMemberRole `gorm:"type:enum('leader','member');default:'member'"`
	JoinedAt time.Time
}

func (TeamMember) TableName() string {
	return "dalictf_team_members"
}
