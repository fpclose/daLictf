// file: models/school.go
package models

import (
	"gorm.io/gorm"
	"time"
)

// 自定义学校状态类型
type SchoolStatus string

const (
	SchoolStatusActive    SchoolStatus = "active"
	SchoolStatusSuspended SchoolStatus = "suspended"
)

type School struct {
	ID             uint32         `gorm:"primarykey" json:"id"`
	SchoolName     string         `gorm:"column:school_name;size:100;unique;not null" json:"school_name"`
	InvitationCode string         `gorm:"size:20;unique;not null" json:"invitation_code"`
	UserCount      uint32         `gorm:"default:0" json:"user_count"`
	Status         SchoolStatus   `gorm:"type:enum('active','suspended');default:'active';not null" json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"` // GORM 的软删除支持, JSON中忽略
}

// TableName 方法告诉 GORM 这个模型对应的表名
func (School) TableName() string {
	return "dalictf_school"
}

// 定义公开和管理员两种不同的序列化结构
type PublicSchoolInfo struct {
	ID         uint32 `json:"id"`
	SchoolName string `json:"school_name"`
	UserCount  uint32 `json:"user_count"`
}

type AdminSchoolInfo struct {
	ID             uint32       `json:"id"`
	SchoolName     string       `json:"school_name"`
	InvitationCode string       `json:"invitation_code"`
	UserCount      uint32       `json:"user_count"`
	Status         SchoolStatus `json:"status"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}
