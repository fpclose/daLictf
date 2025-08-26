// file: models/attachment.go
package models

import (
	"time"
)

type AttachmentStorage string
type AttachmentStatus string
type AttachmentVisibility string

const (
	StorageURL    AttachmentStorage = "url"
	StorageObject AttachmentStorage = "object"

	// =======================================================
	//  ↓↓↓ 修正点：为常量添加 "Attachment" 前缀以避免重名 ↓↓↓
	// =======================================================
	AttachmentStatusPendingScan AttachmentStatus = "pending_scan"
	AttachmentStatusActive      AttachmentStatus = "active"
	AttachmentStatusQuarantined AttachmentStatus = "quarantined"
	AttachmentStatusArchived    AttachmentStatus = "archived"
	// =======================================================

	VisibilityPrivate AttachmentVisibility = "private"
	VisibilityPublic  AttachmentVisibility = "public"
)

type Attachment struct {
	ID           uint64               `gorm:"primarykey"`
	ChallengeID  uint32               `gorm:"not null"`
	Storage      AttachmentStorage    `gorm:"type:enum('url','object');not null"`
	URL          string               `gorm:"size:2048"`
	ObjectBucket string               `gorm:"size:63"`
	ObjectKey    string               `gorm:"size:512"`
	FileName     string               `gorm:"size:255;not null"`
	ContentType  string               `gorm:"size:255;not null"`
	FileSize     uint64               `gorm:"default:0"`
	SHA256       string               `gorm:"size:64;not null"`
	Status       AttachmentStatus     `gorm:"type:enum('pending_scan','active','quarantined','archived');default:'pending_scan'"`
	Visibility   AttachmentVisibility `gorm:"type:enum('private','public');default:'private'"`
	Version      uint16               `gorm:"default:1"`
	SortOrder    uint                 `gorm:"default:0"`
	CreatedBy    uint32               `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Attachment) TableName() string {
	return "dalictf_attachment"
}
