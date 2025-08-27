// file: controllers/challenge_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/dto"
	"ISCTF/models"
	"ISCTF/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"strconv"
	"strings"
)

// CreateChallenge —— 使用 DTO + 手动映射 + Normalize 兼容
func CreateChallenge(c *gin.Context) {
	var req dto.CreateChallengeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}
	req.Normalize() // 兼容 camelCase / snake_case

	// 必填校验（统一在这里做，避免绑定阶段因别名导致的校验失败）
	if req.ChallengeName == "" || req.ChallengeTypeID == 0 || req.Author == "" ||
		req.Description == "" || req.Mode == "" || req.InitialScore == 0 {
		utils.Error(c, 1001, "缺少必填字段")
		return
	}
	if req.Mode != "static" && req.Mode != "dynamic" {
		utils.Error(c, 1001, "mode 取值无效（static/dynamic）")
		return
	}
	if req.Mode == "static" && strings.TrimSpace(req.StaticFlag) == "" {
		utils.Error(c, 1002, "静态题目必须提供 Flag")
		return
	}
	if req.Mode == "dynamic" && strings.TrimSpace(req.DockerImage) == "" {
		utils.Error(c, 1002, "动态题目必须提供 Docker 镜像")
		return
	}
	if req.MinScore > req.InitialScore {
		utils.Error(c, 1001, "min_score 不能大于 initial_score")
		return
	}
	if req.Difficulty != "" && req.Difficulty != "easy" && req.Difficulty != "medium" && req.Difficulty != "hard" {
		utils.Error(c, 1001, "difficulty 取值无效（easy/medium/hard）")
		return
	}

	// 题目类型存在性校验
	var qt models.QuestionType
	if err := database.DB.First(&qt, req.ChallengeTypeID).Error; err != nil {
		utils.Error(c, 4001, "题目类型不存在")
		return
	}

	// 手动映射到模型
	chal := models.Challenge{
		ChallengeName:   req.ChallengeName,
		ChallengeTypeID: req.ChallengeTypeID,
		Author:          req.Author,
		Description:     req.Description,
		Hint:            req.Hint,
		Mode:            models.ChallengeMode(req.Mode),
		StaticFlag:      req.StaticFlag,
		DockerImage:     req.DockerImage,
		DockerPorts:     req.DockerPorts,
		Difficulty:      models.ChallengeDifficulty(req.Difficulty),
		InitialScore:    req.InitialScore,
		MinScore:        req.MinScore,
		CurrentScore:    req.InitialScore, // 初始化为初始分
		DecayRatio:      req.DecayRatio,
	}

	if err := database.DB.Create(&chal).Error; err != nil {
		utils.Error(c, 5000, "创建题目失败: "+err.Error())
		return
	}
	utils.Success(c, "Challenge created successfully", gin.H{"id": chal.ID})
}

// ListChallenges —— 用户可见的题目列表
func ListChallenges(c *gin.Context) {
	var challenges []models.Challenge
	db := database.DB.Model(&models.Challenge{}).
		Where("state = ?", models.ChallengeStateVisible).
		Preload("QuestionType")

	// TODO: 可在此读取 query 参数增加筛选

	if err := db.Find(&challenges).Error; err != nil {
		utils.Error(c, 5000, "查询失败")
		return
	}

	items := make([]dto.ChallengeItemResp, 0, len(challenges))
	for _, ch := range challenges {
		items = append(items, dto.ChallengeItemResp{
			ID:            ch.ID,
			ChallengeName: ch.ChallengeName,
			Type:          ch.QuestionType.Alias,
			Difficulty:    string(ch.Difficulty),
			Mode:          string(ch.Mode),
			CurrentScore:  ch.CurrentScore,
			SolvedCount:   ch.SolvedCount,
		})
	}

	utils.Success(c, "success", gin.H{
		"total":      len(items),
		"challenges": items,
	})
}

// GetChallengeDetail —— 用户可见的题目详情
func GetChallengeDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var challenge models.Challenge
	if err := database.DB.Preload("QuestionType").First(&challenge, id).Error; err != nil {
		utils.Error(c, 4004, "题目不存在")
		return
	}
	if challenge.State != models.ChallengeStateVisible {
		utils.Error(c, 4003, "题目不可见")
		return
	}

	var attachments []models.Attachment
	if err := database.DB.
		Where("challenge_id = ? AND status = ?", id, models.AttachmentStatusActive).
		Find(&attachments).Error; err != nil {
		utils.Error(c, 5000, "附件查询失败")
		return
	}

	mini := make([]dto.AttachmentMini, 0, len(attachments))
	for _, a := range attachments {
		mini = append(mini, dto.AttachmentMini{
			ID:       a.ID,
			FileName: a.FileName,
			Size:     uint64(a.FileSize),
			SHA256:   a.SHA256,
			Status:   string(a.Status),
		})
	}

	resp := dto.ChallengeDetailResp{
		ID:            challenge.ID,
		ChallengeName: challenge.ChallengeName,
		Author:        challenge.Author,
		Description:   challenge.Description,
		Hint:          challenge.Hint,
		Mode:          string(challenge.Mode),
		Difficulty:    string(challenge.Difficulty),
		Attachments:   mini,
		CurrentScore:  challenge.CurrentScore,
		SolvedCount:   challenge.SolvedCount,
	}

	utils.Success(c, "success", resp)
}

// SubmitFlag —— 提交 Flag 并处理分数衰减、队伍判重等
func SubmitFlag(c *gin.Context) {
	challengeID, _ := strconv.Atoi(c.Param("id"))

	var req dto.SubmitFlagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}
	req.Normalize()

	// 从中间件读取用户信息
	userIDAny, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 4001, "未登录")
		return
	}
	userID := userIDAny.(uint32)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 必须加入队伍
		var userTeam models.TeamMember
		if err := tx.Where("user_id = ?", userID).First(&userTeam).Error; err != nil {
			return errors.New("你尚未加入队伍")
		}

		// 对题目行加锁，避免并发更新
		var challenge models.Challenge
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&challenge, challengeID).Error; err != nil {
			return errors.New("题目不存在")
		}

		// 同队伍重复解题判定
		var existingSubmission models.Submission
		tx.Where("challenge_id = ? AND team_id = ?", challengeID, userTeam.TeamID).
			First(&existingSubmission)
		if existingSubmission.ID != 0 {
			return errors.New("你所在的队伍已解出此题")
		}

		// 静态题校验 Flag（动态题另行实现）
		if challenge.Mode != models.ChallengeModeStatic || challenge.StaticFlag != req.Flag {
			return errors.New("Flag 错误")
		}

		// 写入提交记录
		submission := models.Submission{
			ChallengeID: uint32(challengeID),
			UserID:      userID,
			TeamID:      userTeam.TeamID,
			IsCorrect:   true,
		}
		if err := tx.Create(&submission).Error; err != nil {
			return err
		}

		// 分数衰减：每解出一次衰减 initial_score * decay_ratio（至少 1 分）
		challenge.SolvedCount++
		decay := uint(math.Round(float64(challenge.InitialScore) * float64(challenge.DecayRatio)))
		if decay == 0 && challenge.DecayRatio > 0 {
			decay = 1
		}
		newScore := int(challenge.CurrentScore) - int(decay)
		if newScore < int(challenge.MinScore) {
			newScore = int(challenge.MinScore)
		}
		challenge.CurrentScore = uint(newScore)

		return tx.Save(&challenge).Error
	})

	if err != nil {
		utils.Error(c, 5001, err.Error())
		return
	}

	utils.Success(c, "Flag 正确！", gin.H{})
}

// AdminListChallenges —— 管理员查询题目列表（可见/隐藏均可，支持筛选+分页）
func AdminListChallenges(c *gin.Context) {
	// 读取筛选参数
	typeIDStr := c.Query("type_id")
	mode := strings.TrimSpace(c.Query("mode"))       // static/dynamic
	diff := strings.TrimSpace(c.Query("difficulty")) // easy/medium/hard
	state := strings.TrimSpace(c.Query("state"))     // visible/hidden
	kw := strings.TrimSpace(c.Query("keyword"))      // 模糊匹配 name/description
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	db := database.DB.Model(&models.Challenge{}).Preload("QuestionType")

	if typeIDStr != "" {
		if tid, err := strconv.Atoi(typeIDStr); err == nil && tid > 0 {
			db = db.Where("challenge_type_id = ?", tid)
		}
	}
	if mode != "" {
		db = db.Where("mode = ?", models.ChallengeMode(mode))
	}
	if diff != "" {
		db = db.Where("difficulty = ?", models.ChallengeDifficulty(diff))
	}
	if state != "" {
		db = db.Where("state = ?", models.ChallengeState(state))
	}
	if kw != "" {
		like := "%" + kw + "%"
		db = db.Where("challenge_name LIKE ? OR description LIKE ?", like, like)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		utils.Error(c, 5000, "查询失败: "+err.Error())
		return
	}

	var list []models.Challenge
	if err := db.Order("updated_at DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		utils.Error(c, 5000, "查询失败: "+err.Error())
		return
	}

	items := make([]dto.AdminChallengeItemResp, 0, len(list))
	for _, ch := range list {
		items = append(items, dto.AdminChallengeItemResp{
			ID:            ch.ID,
			ChallengeName: ch.ChallengeName,
			Type:          ch.QuestionType.Alias,
			Difficulty:    string(ch.Difficulty),
			Mode:          string(ch.Mode),
			State:         string(ch.State),
			CurrentScore:  ch.CurrentScore,
			SolvedCount:   ch.SolvedCount,
			UpdatedAt:     ch.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.Success(c, "success", gin.H{
		"total":      total,
		"page":       page,
		"limit":      limit,
		"challenges": items,
	})
}

// AdminGetChallengeDetail —— 管理员查询题目详情（不受可见性限制，附件返回所有状态）
func AdminGetChallengeDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var ch models.Challenge
	if err := database.DB.Preload("QuestionType").First(&ch, id).Error; err != nil {
		utils.Error(c, 4004, "题目不存在")
		return
	}

	var atts []models.Attachment
	if err := database.DB.
		Where("challenge_id = ?", id).
		Order("sort_order ASC, id ASC").
		Find(&atts).Error; err != nil {
		utils.Error(c, 5000, "附件查询失败")
		return
	}

	mini := make([]dto.AdminAttachmentMini, 0, len(atts))
	for _, a := range atts {
		mini = append(mini, dto.AdminAttachmentMini{
			ID:       a.ID,
			FileName: a.FileName,
			Size:     uint64(a.FileSize),
			SHA256:   a.SHA256,
			Status:   string(a.Status),
			Storage:  string(a.Storage),
		})
	}

	resp := dto.AdminChallengeDetailResp{
		ID:            ch.ID,
		ChallengeName: ch.ChallengeName,
		Type:          ch.QuestionType.Alias,
		Author:        ch.Author,
		Description:   ch.Description,
		Hint:          ch.Hint,
		Mode:          string(ch.Mode),
		Difficulty:    string(ch.Difficulty),
		State:         string(ch.State),
		StaticFlag:    ch.StaticFlag,
		DockerImage:   ch.DockerImage,
		DockerPorts:   ch.DockerPorts,
		CurrentScore:  ch.CurrentScore,
		InitialScore:  ch.InitialScore,
		MinScore:      ch.MinScore,
		DecayRatio:    ch.DecayRatio,
		SolvedCount:   ch.SolvedCount,
		Attachments:   mini,
		CreatedAt:     ch.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     ch.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	utils.Success(c, "success", resp)
}
