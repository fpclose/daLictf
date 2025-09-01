// file: controllers/scoreboard_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
)

// GetScoreboard 查询排行榜
func GetScoreboard(c *gin.Context) {
	track := c.DefaultQuery("track", "overall")
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// 1. 尝试从 Redis 获取缓存
	cacheKey := fmt.Sprintf("scoreboard:%s:%d", track, limit)
	val, err := database.RDB.Get(database.Ctx, cacheKey).Result()
	if err == nil {
		var results []models.Scoreboard
		if json.Unmarshal([]byte(val), &results) == nil {
			utils.Success(c, "success (from cache)", results)
			return
		}
	}

	var results []models.Scoreboard
	// 修正：为保留字 rank 加上反引号
	database.DB.Where("track = ?", track).Order("`rank` asc").Limit(limit).Find(&results)

	// 2. 如果缓存未命中，则将数据库查询结果存入 Redis
	jsonData, err := json.Marshal(results)
	if err == nil {
		// 将缓存有效期设置为较短的15秒，以保证排行榜的准实时性
		database.RDB.Set(database.Ctx, cacheKey, jsonData, 15*time.Second)
	}

	utils.Success(c, "success", results)
}

// GetSolveFeed 查询实时解题动态
func GetSolveFeed(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var results []models.SolveFeed
	database.DB.Order("solving_time desc").Limit(limit).Find(&results)

	utils.Success(c, "success", results)
}
