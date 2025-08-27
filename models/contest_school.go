// file: models/contest_school.go
package models

// ContestSchool 对应 dalictf_contest_schools 关联表 (已添加 JSON 标签)
type ContestSchool struct {
	ID         uint   `gorm:"primarykey" json:"id,omitempty"`
	ContestID  uint   `gorm:"not null" json:"contest_id"`
	SchoolID   uint32 `gorm:"not null" json:"school_id"`
	SchoolLogo string `gorm:"size:255" json:"school_logo"`
}

func (ContestSchool) TableName() string {
	return "dalictf_contest_schools"
}
