// file: main.go
package main

import (
	"ISCTF/database"
	"ISCTF/routes"
	"log"
)

func main() {
	database.Connect()

	// 禁用自动迁移 (推荐)
	// database.MigrateTables()

	// 如果选择使用自动迁移，需要更新 MigrateTables 函数
	// 在 database/connect.go 中:
	// func MigrateTables() {
	// 	err := DB.AutoMigrate(&models.School{}, &models.User{}, &models.Team{}, &models.TeamMember{})
	// 	...
	// }

	r := routes.SetupRouter()

	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
