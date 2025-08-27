// file: models/contest_sponsor.go
package models

// ContestSponsor 对应 dalictf_contest_sponsors 表 (已添加 JSON 标签)
type ContestSponsor struct {
	ID          uint   `gorm:"primarykey" json:"id,omitempty"`
	ContestID   uint   `gorm:"not null" json:"contest_id"`
	SponsorName string `gorm:"size:100;not null" json:"sponsor_name"`
	LogoURL     string `gorm:"size:255" json:"logo_url"`
	Description string `gorm:"type:text" json:"description"`
	Link        string `gorm:"size:255" json:"link"`
}

func (ContestSponsor) TableName() string {
	return "dalictf_contest_sponsors"
}
