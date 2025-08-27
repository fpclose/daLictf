// file: controllers/contest_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
	"strconv"
	"time"
)

// GetCurrentContest 查询当前比赛基本信息
func GetCurrentContest(c *gin.Context) {
	var contest models.Contest
	// 假设我们总是查询 ID 为 1 的比赛作为当前比赛
	if err := database.DB.First(&contest, 1).Error; err != nil {
		utils.Error(c, 404, "No active contest found")
		return
	}

	// 查询关联的学校
	type SchoolInfo struct {
		SchoolName string `json:"school_name"`
		SchoolLogo string `json:"school_logo"`
	}
	var schools []SchoolInfo
	database.DB.Table("dalictf_contest_schools cs").
		Select("s.school_name, cs.school_logo").
		Joins("JOIN dalictf_school s ON cs.school_id = s.id").
		Where("cs.contest_id = ?", contest.ID).
		Find(&schools)

	// 查询关联的赞助商
	var sponsors []models.ContestSponsor
	database.DB.Where("contest_id = ?", contest.ID).Find(&sponsors)

	// 自动计算并更新比赛状态
	now := time.Now()
	var currentStatus models.ContestStatus
	if now.Before(contest.StartTime) {
		currentStatus = models.ContestStatusPreparing
	} else if now.After(contest.EndTime) {
		currentStatus = models.ContestStatusEnded
	} else {
		currentStatus = models.ContestStatusRunning
	}

	utils.Success(c, "success", gin.H{
		"contest_id":    contest.ID,
		"contest_name":  contest.ContestName,
		"cover_image":   contest.CoverImage,
		"description":   contest.Description,
		"start_time":    contest.StartTime.Format("2006-01-02 15:04:05"),
		"end_time":      contest.EndTime.Format("2006-01-02 15:04:05"),
		"organizer_url": contest.OrganizerURL,
		"status":        currentStatus,
		"schools":       schools,
		"sponsors":      sponsors,
	})
}

// GetContestStatus 查询比赛状态和剩余时间
func GetContestStatus(c *gin.Context) {
	var contest models.Contest
	if err := database.DB.First(&contest, 1).Error; err != nil {
		utils.Error(c, 404, "No active contest found")
		return
	}

	now := time.Now()
	var status models.ContestStatus
	var remainingTime string

	if now.Before(contest.StartTime) {
		status = models.ContestStatusPreparing
		remainingTime = contest.StartTime.Sub(now).Round(time.Second).String()
	} else if now.After(contest.EndTime) {
		status = models.ContestStatusEnded
		remainingTime = "0s"
	} else {
		status = models.ContestStatusRunning
		remainingTime = contest.EndTime.Sub(now).Round(time.Second).String()
	}

	utils.Success(c, "success", gin.H{
		"status":         status,
		"now":            now.Format("2006-01-02 15:04:05"),
		"remaining_time": remainingTime,
	})
}

// --- 管理员接口 ---

// UpsertContest 创建或修改比赛信息
func UpsertContest(c *gin.Context) {
	var req models.Contest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	// 使用 GORM 的 Upsert 功能，存在则更新，不存在则创建 (ID=1)
	req.ID = 1
	if err := database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"contest_name", "cover_image", "description", "start_time", "end_time", "organizer_url"}),
	}).Create(&req).Error; err != nil {
		utils.Error(c, 5000, "Failed to create/update contest: "+err.Error())
		return
	}

	utils.Success(c, "Contest created/updated successfully", nil)
}

// AddContestSchool 为比赛添加参赛学校
func AddContestSchool(c *gin.Context) {
	var req models.ContestSchool
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	if err := database.DB.Create(&req).Error; err != nil {
		utils.Error(c, 5000, "Failed to add school to contest: "+err.Error())
		return
	}
	utils.Success(c, "School added to contest successfully", gin.H{"id": req.ID})
}

// DeleteContestSchool 从比赛中移除参赛学校
func DeleteContestSchool(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := database.DB.Delete(&models.ContestSchool{}, id).Error; err != nil {
		utils.Error(c, 5000, "Failed to delete school from contest: "+err.Error())
		return
	}
	utils.Success(c, "School deleted from contest successfully", nil)
}

// AddContestSponsor 为比赛添加赞助商
func AddContestSponsor(c *gin.Context) {
	var req models.ContestSponsor
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	if err := database.DB.Create(&req).Error; err != nil {
		utils.Error(c, 5000, "Failed to add sponsor to contest: "+err.Error())
		return
	}
	utils.Success(c, "Sponsor added to contest successfully", gin.H{"id": req.ID})
}

// DeleteContestSponsor 从比赛中移除赞助商
func DeleteContestSponsor(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := database.DB.Delete(&models.ContestSponsor{}, id).Error; err != nil {
		utils.Error(c, 5000, "Failed to delete sponsor from contest: "+err.Error())
		return
	}
	utils.Success(c, "Sponsor deleted from contest successfully", nil)
}
