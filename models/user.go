// file: models/user.go
package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"time"
)

// 自定义类型 UserTrack, UserRole, UserStatus
type UserTrack string
type UserRole string
type UserStatus string

const (
	TrackFreshman UserTrack  = "freshman"
	TrackAdvanced UserTrack  = "advanced"
	TrackSociety  UserTrack  = "society"
	RoleUser      UserRole   = "user"
	RoleAdmin     UserRole   = "admin"
	RoleRootAdmin UserRole   = "root_admin"
	StatusActive  UserStatus = "active"
	StatusBanned  UserStatus = "banned"
)

type User struct {
	ID            uint32     `gorm:"primarykey" json:"id"`
	Username      string     `gorm:"size:50;unique;not null" json:"username"`
	Password      string     `gorm:"size:255;not null" json:"-"`
	Email         string     `gorm:"size:100;unique;not null" json:"email"`
	RealName      string     `gorm:"size:50" json:"real_name,omitempty"`
	SchoolID      *uint32    `json:"school_id,omitempty"`
	School        *School    `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	StudentNumber string     `gorm:"size:50" json:"student_number,omitempty"`
	GradeYear     *int       `gorm:"type:year" json:"grade_year,omitempty"`
	Track         UserTrack  `gorm:"type:enum('freshman','advanced','society');not null;default:'society'" json:"track"`
	Role          UserRole   `gorm:"type:enum('user','admin','root_admin');not null;default:'user'" json:"role"`
	Status        UserStatus `gorm:"type:enum('active','banned');not null;default:'active'" json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (User) TableName() string {
	return "dalictf_user"
}

// BeforeSave GORM Hook，在保存用户前自动哈希密码
func (u *User) BeforeSave(tx *gorm.DB) (err error) {
	// 在新用户创建时 (ID=0) 或在老用户更新密码时，都执行哈希
	if u.ID == 0 || tx.Statement.Changed("Password") {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return
}

// CheckPassword 校验密码是否正确
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
