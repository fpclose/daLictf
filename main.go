// file: main.go
package main

import (
	"ISCTF/database"
	"ISCTF/routes"
	"ISCTF/services" // 确保导入 services 包
	"log"
)

func main() {
	// 1. 连接数据库
	database.Connect()

	// 2. 初始化 Redis 客户端
	database.InitRedis()

	// 3. 初始化 Docker 客户端
	services.InitDocker()

	//// 4. 自动迁移数据库表结构
	//database.MigrateTables()

	// 5. 设置并获取路由引擎
	r := routes.SetupRouter()

	// 6. 启动服务器
	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
