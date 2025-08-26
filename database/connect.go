// file: database/connect.go
package database

import (
	"ISCTF/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"time"
)

var DB *gorm.DB

func Connect() {
	var err error
	dsn := "root:123456@tcp(localhost:3306)/dali_isctf?charset=utf8mb4&parseTime=True&loc=Local"
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// =======================================================
	//  ↓↓↓ 新增：配置数据库连接池 ↓↓↓
	// =======================================================
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get underlying sql.DB:", err)
	}

	// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	// 这对于解决 MySQL 的 'wait_timeout' 问题至关重要。
	// 设置为1小时，意味着连接在创建1小时后会被标记为过期，
	// GORM 在下次使用它之前会安全地重新建立连接。
	sqlDB.SetConnMaxLifetime(time.Hour)
	// =======================================================

	log.Println("Database connection successfully established and connection pool configured.")
}

// MigrateTables 函数 (如果你不希望 GORM 自动修改表结构，也应该禁用它)
func MigrateTables() {
	err := DB.AutoMigrate(&models.School{}, &models.User{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	log.Println("Database migration completed.")
}
