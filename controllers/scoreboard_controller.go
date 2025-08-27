// file: controllers/scoreboard_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"github.com/gin-gonic/gin"
	"strconv"
)

// GetScoreboard 查询排行榜
func GetScoreboard(c *gin.Context) {
	track := c.DefaultQuery("track", "overall")
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	var results []models.Scoreboard
	// 修正：为保留字 rank 加上反引号
	database.DB.Where("track = ?", track).Order("`rank` asc").Limit(limit).Find(&results)

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
