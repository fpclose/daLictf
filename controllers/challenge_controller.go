// file: controllers/challenge_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"strconv"
)

func CreateChallenge(c *gin.Context) {
	var req models.Challenge
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	if req.Mode == models.ChallengeModeStatic && req.StaticFlag == "" {
		utils.Error(c, 1002, "静态题目必须提供 Flag")
		return
	}
	if req.Mode == models.ChallengeModeDynamic && req.DockerImage == "" {
		utils.Error(c, 1002, "动态题目必须提供 Docker 镜像")
		return
	}

	req.CurrentScore = req.InitialScore

	if err := database.DB.Create(&req).Error; err != nil {
		utils.Error(c, 5000, "创建题目失败: "+err.Error())
		return
	}

	utils.Success(c, "Challenge created successfully", gin.H{"id": req.ID})
}

func ListChallenges(c *gin.Context) {
	var challenges []models.Challenge
	db := database.DB.Model(&models.Challenge{}).
		Where("state = ?", models.ChallengeStateVisible).
		Preload("QuestionType")
	db.Find(&challenges)

	type ChallengeDTO struct {
		ID            uint32                     `json:"id"`
		ChallengeName string                     `json:"challenge_name"`
		Type          string                     `json:"type"`
		Difficulty    models.ChallengeDifficulty `json:"difficulty"`
		Mode          models.ChallengeMode       `json:"mode"`
		CurrentScore  uint                       `json:"current_score"`
		SolvedCount   uint                       `json:"solved_count"`
	}

	result := make([]ChallengeDTO, len(challenges))
	for i, ch := range challenges {
		result[i] = ChallengeDTO{
			ID:            ch.ID,
			ChallengeName: ch.ChallengeName,
			Type:          ch.QuestionType.Alias,
			Difficulty:    ch.Difficulty,
			Mode:          ch.Mode,
			CurrentScore:  ch.CurrentScore,
			SolvedCount:   ch.SolvedCount,
		}
	}

	utils.Success(c, "success", gin.H{
		"total":      len(result),
		"challenges": result,
	})
}

func GetChallengeDetail(c *gin.Context) {
	challengeID, _ := strconv.Atoi(c.Param("id"))
	var challenge models.Challenge

	err := database.DB.
		Preload("QuestionType").
		Preload("Attachments", "status = ?", models.AttachmentStatusActive). // 使用修正后的常量
		Where("state = ?", models.ChallengeStateVisible).
		First(&challenge, challengeID).Error

	if err != nil {
		utils.Error(c, 4004, "题目不存在或已隐藏")
		return
	}

	type AttachmentDTO struct {
		ID       uint64 `json:"id"`
		FileName string `json:"file_name"`
		Size     uint64 `json:"size"`
		SHA256   string `json:"sha256"`
	}

	attachments := make([]AttachmentDTO, len(challenge.Attachments))
	for i, att := range challenge.Attachments {
		attachments[i] = AttachmentDTO{
			ID:       att.ID,
			FileName: att.FileName,
			Size:     att.FileSize,
			SHA256:   att.SHA256,
		}
	}

	utils.Success(c, "success", gin.H{
		"id":             challenge.ID,
		"challenge_name": challenge.ChallengeName,
		"type":           challenge.QuestionType.Alias,
		"author":         challenge.Author,
		"description":    challenge.Description,
		"hint":           challenge.Hint,
		"mode":           challenge.Mode,
		"difficulty":     challenge.Difficulty,
		"attachments":    attachments,
		"current_score":  challenge.CurrentScore,
		"solved_count":   challenge.SolvedCount,
	})
}

// SubmitFlag 提交Flag并处理分数衰减
func SubmitFlag(c *gin.Context) {
	challengeID, _ := strconv.Atoi(c.Param("id"))

	// =======================================================
	//  ↓↓↓ 修正点：使用兼容旧版Gin的写法 ↓↓↓
	// =======================================================
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint32)
	// =======================================================

	var req struct {
		Flag string `json:"flag" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效")
		return
	}

	var userTeam models.TeamMember
	database.DB.Where("user_id = ?", userID).First(&userTeam)
	if userTeam.TeamID == 0 {
		utils.Error(c, 3001, "请先加入队伍")
		return
	}

	var challenge models.Challenge
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&challenge, challengeID).Error; err != nil {
			return errors.New("题目不存在")
		}

		var existingSubmission models.Submission
		tx.Where("challenge_id = ? AND team_id = ?", challengeID, userTeam.TeamID).First(&existingSubmission)
		if existingSubmission.ID != 0 {
			return errors.New("你所在的队伍已解出此题")
		}

		if challenge.Mode != models.ChallengeModeStatic || challenge.StaticFlag != req.Flag {
			return errors.New("Flag 错误")
		}

		submission := models.Submission{
			ChallengeID: uint32(challengeID),
			UserID:      userID,
			TeamID:      userTeam.TeamID,
			IsCorrect:   true,
		}
		if err := tx.Create(&submission).Error; err != nil {
			return err
		}

		challenge.SolvedCount++
		newScore := float64(challenge.InitialScore) - float64(challenge.SolvedCount)*float64(challenge.DecayRatio)*float64(challenge.InitialScore)
		challenge.CurrentScore = uint(math.Max(float64(challenge.MinScore), newScore))

		if err := tx.Save(&challenge).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		utils.Error(c, 5001, err.Error())
		return
	}

	utils.Success(c, "Flag 正确！", gin.H{
		"new_score": challenge.CurrentScore,
	})
}
