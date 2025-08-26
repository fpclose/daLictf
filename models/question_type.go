// file: models/question_type.go
package models

import (
	"time"
)

type QuestionType struct {
	ID          uint32    `gorm:"primarykey" json:"id"`
	Direction   string    `gorm:"size:50;unique;not null" json:"direction"`
	Alias       string    `gorm:"size:50" json:"alias"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (QuestionType) TableName() string {
	return "dalictf_question_type"
}
