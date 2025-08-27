// file: controllers/record_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"errors" // <-- 新增：导入标准库 errors
	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // <-- 新增：导入 gorm 核心包
	"strconv"
	"time"
)

// GetTeamSolves 查询队伍解题记录
func GetTeamSolves(c *gin.Context) {
	teamID, _ := strconv.Atoi(c.Param("id"))

	var solves []models.Submission
	database.DB.Where("team_id = ?", teamID).Order("solving_time asc").Find(&solves)

	type SolveInfo struct {
		ChallengeID   uint32 `json:"challenge_id"`
		ChallengeName string `json:"challenge_name"`
		Score         uint   `json:"score"`
		SolvingTime   string `json:"solving_time"`
	}
	var result []SolveInfo
	for _, solve := range solves {
		var chal models.Challenge
		database.DB.Select("challenge_name").First(&chal, solve.ChallengeID)
		result = append(result, SolveInfo{
			ChallengeID:   solve.ChallengeID,
			ChallengeName: chal.ChallengeName,
			Score:         solve.Score,
			SolvingTime:   solve.SolvingTime.Format("2006-01-02 15:04:05"),
		})
	}

	utils.Success(c, "success", result)
}

// GetFlagLogs 管理员查询 Flag 提交日志
func GetFlagLogs(c *gin.Context) {
	type LogDetail struct {
		ID             uint64    `json:"id"`
		ChallengeID    uint32    `json:"challenge_id"`
		ChallengeName  string    `json:"challenge_name"`
		TeamID         uint32    `json:"team_id"`
		TeamName       string    `json:"team_name"`
		UserID         uint32    `json:"user_id"`
		Username       string    `json:"username"`
		SubmittedFlag  string    `json:"submitted_flag"`
		FlagResult     string    `json:"flag_result"`
		SubmissionTime time.Time `json:"submission_time"`
		IPAddress      string    `json:"ip_address"`
		Suspected      bool      `json:"suspected"`
	}

	db := database.DB.Table("dalictf_flag_information l").
		Select("l.id, l.challenge_id, c.challenge_name, l.team_id, t.team_name, l.user_id, u.username, l.submitted_flag, l.flag_result, l.submission_time, l.ip_address, l.suspected").
		Joins("LEFT JOIN dalictf_challenge c ON l.challenge_id = c.id").
		Joins("LEFT JOIN dalictf_team t ON l.team_id = t.id").
		Joins("LEFT JOIN dalictf_user u ON l.user_id = u.id")

	if teamID := c.Query("team_id"); teamID != "" {
		db = db.Where("l.team_id = ?", teamID)
	}
	if challengeID := c.Query("challenge_id"); challengeID != "" {
		db = db.Where("l.challenge_id = ?", challengeID)
	}
	if userID := c.Query("user_id"); userID != "" {
		db = db.Where("l.user_id = ?", userID)
	}
	if result := c.Query("result"); result != "" {
		db = db.Where("l.flag_result = ?", result)
	}
	if suspected := c.Query("suspected"); suspected == "1" {
		db = db.Where("l.suspected = ?", true)
	}

	var results []LogDetail
	db.Order("l.submission_time desc").Find(&results)

	utils.Success(c, "success", results)
}

// MarkSuspectSubmission 管理员手动标记可疑提交
func MarkSuspectSubmission(c *gin.Context) {
	logID, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		Suspected bool `json:"suspected"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "Invalid request body")
		return
	}

	result := database.DB.Model(&models.SubmissionLog{}).Where("id = ?", logID).Update("suspected", req.Suspected)
	if result.Error != nil {
		utils.Error(c, 5000, "Database update failed: "+result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		utils.Error(c, 404, "Submission log not found")
		return
	}

	utils.Success(c, "Flag submission marked as suspected", nil)
}

// CompareFlagSubmissions 对比相同 flag 的提交记录
func CompareFlagSubmissions(c *gin.Context) {
	flag := c.Query("flag")
	if flag == "" {
		utils.Error(c, 1001, "Missing 'flag' query parameter")
		return
	}

	var firstSubmission models.SubmissionLog
	err := database.DB.Where("submitted_flag = ?", flag).First(&firstSubmission).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Error(c, 404, "No submissions found for this flag")
			return
		}
		utils.Error(c, 5000, "Database error")
		return
	}

	var challenge models.Challenge
	database.DB.First(&challenge, firstSubmission.ChallengeID)
	if challenge.Mode != models.ChallengeModeDynamic {
		utils.Error(c, 400, "Comparison is only applicable for dynamic challenges")
		return
	}

	type CompareResult struct {
		TeamID         uint32    `json:"team_id"`
		TeamName       string    `json:"team_name"`
		SchoolName     *string   `json:"school_name"`
		UserID         uint32    `json:"user_id"`
		Username       string    `json:"username"`
		SubmissionTime time.Time `json:"submission_time"`
	}

	var results []CompareResult
	database.DB.Table("dalictf_flag_information l").
		Select("l.team_id, t.team_name, s.school_name, l.user_id, u.username, l.submission_time").
		Joins("JOIN dalictf_team t ON l.team_id = t.id").
		Joins("JOIN dalictf_user u ON l.user_id = u.id").
		Joins("LEFT JOIN dalictf_school s ON t.school_id = s.id").
		Where("l.submitted_flag = ?", flag).
		Order("l.submission_time asc").
		Find(&results)

	utils.Success(c, "success", gin.H{
		"flag_value":  flag,
		"submissions": results,
	})
}
