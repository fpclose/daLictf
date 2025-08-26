// file: models/challenge.go
package models

import (
	"time"
)

type ChallengeState string
type ChallengeMode string
type ChallengeDifficulty string

const (
	ChallengeStateVisible ChallengeState = "visible"
	ChallengeStateHidden  ChallengeState = "hidden"

	ChallengeModeStatic  ChallengeMode = "static"
	ChallengeModeDynamic ChallengeMode = "dynamic"

	ChallengeDifficultyEasy   ChallengeDifficulty = "easy"
	ChallengeDifficultyMedium ChallengeDifficulty = "medium"
	ChallengeDifficultyHard   ChallengeDifficulty = "hard"
)

type Challenge struct {
	ID              uint32              `gorm:"primarykey"`
	ChallengeName   string              `gorm:"size:100;unique;not null"`
	ChallengeTypeID uint32              `gorm:"not null"`
	QuestionType    QuestionType        `gorm:"foreignKey:ChallengeTypeID"`
	Author          string              `gorm:"size:50;not null"`
	Description     string              `gorm:"type:text;not null"`
	Hint            string              `gorm:"type:text"`
	State           ChallengeState      `gorm:"type:enum('visible','hidden');default:'hidden'"`
	Mode            ChallengeMode       `gorm:"type:enum('static','dynamic');not null"`
	StaticFlag      string              `gorm:"size:255"`
	DockerImage     string              `gorm:"size:255"`
	DockerPorts     string              `gorm:"size:50"`
	Difficulty      ChallengeDifficulty `gorm:"type:enum('easy','medium','hard');default:'medium'"`
	InitialScore    uint                `gorm:"not null"`
	MinScore        uint                `gorm:"not null"`
	CurrentScore    uint                `gorm:"not null"`
	DecayRatio      float32             `gorm:"default:0.1"`
	SolvedCount     uint                `gorm:"default:0"`
	Attachments     []Attachment        `gorm:"foreignKey:ChallengeID"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (Challenge) TableName() string {
	return "dalictf_challenge"
}
