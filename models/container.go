// file: models/container.go
package models

import (
	"time"
)

type ContainerState string

const (
	ContainerStateRunning   ContainerState = "running"
	ContainerStateStopped   ContainerState = "stopped"
	ContainerStateDestroyed ContainerState = "destroyed"
)

// Container 对应 dalictf_container 表
type Container struct {
	ID             uint32         `gorm:"primarykey"`
	DockerID       string         `gorm:"size:64;not null"`
	ChallengeID    uint32         `gorm:"not null"`
	TeamID         uint32         `gorm:"not null"`
	ContainerName  string         `gorm:"size:100;not null"`
	DockerImage    string         `gorm:"size:255;not null"`
	DockerPorts    string         `gorm:"size:100;not null"`
	ContainerFlag  string         `gorm:"size:255;not null"`
	State          ContainerState `gorm:"type:enum('running','stopped','destroyed');default:'running'"`
	StartTime      time.Time      `gorm:"default:CURRENT_TIMESTAMP"`
	EndTime        time.Time      `gorm:"not null"`
	ExtendedCount  uint           `gorm:"default:0"`
	PcapPath       string         `gorm:"size:255"`
	AnalysisResult string         `gorm:"type:text"`
}

func (Container) TableName() string {
	return "dalictf_container"
}
