// file: controllers/challenge_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/dto"
	"ISCTF/models"
	"ISCTF/services"
	"ISCTF/utils"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
)

// CreateChallenge —— 使用 DTO + 手动映射 + Normalize 兼容
func CreateChallenge(c *gin.Context) {
	var req dto.CreateChallengeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}
	req.Normalize()

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

	var qt models.QuestionType
	if err := database.DB.First(&qt, req.ChallengeTypeID).Error; err != nil {
		utils.Error(c, 4001, "题目类型不存在")
		return
	}

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
		CurrentScore:    req.InitialScore,
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
	cacheKey := "challenge_detail:" + strconv.Itoa(id)

	// 1. 尝试从 Redis 获取缓存
	val, err := database.RDB.Get(database.Ctx, cacheKey).Result()
	if err == nil {
		var resp dto.ChallengeDetailResp
		if json.Unmarshal([]byte(val), &resp) == nil {
			utils.Success(c, "success (from cache)", resp)
			return
		}
	}

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

	// 2. 查询结果存入 Redis，缓存5分钟
	jsonData, err := json.Marshal(resp)
	if err == nil {
		database.RDB.Set(database.Ctx, cacheKey, jsonData, 5*time.Minute)
	}

	utils.Success(c, "success", resp)
}

// SubmitFlag -- 包含完整日志、计分、销毁容器和自动标记逻辑
func SubmitFlag(c *gin.Context) {
	challengeID, _ := strconv.Atoi(c.Param("id"))

	var req dto.SubmitFlagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}
	req.Normalize()

	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint32)

	var userTeam models.TeamMember
	if err := database.DB.Where("user_id = ?", userID).First(&userTeam).Error; err != nil {
		utils.Error(c, 3005, "你尚未加入任何队伍")
		return
	}

	var team models.Team
	database.DB.First(&team, userTeam.TeamID)
	if team.TeamStatus == models.TeamStatusBanned {
		utils.Error(c, 4003, "队伍已被封禁，无法提交 Flag")
		return
	}

	var challenge models.Challenge
	if err := database.DB.First(&challenge, challengeID).Error; err != nil {
		utils.Error(c, 4004, "题目不存在")
		return
	}

	logEntry := models.SubmissionLog{
		ChallengeID:   uint32(challengeID),
		TeamID:        userTeam.TeamID,
		UserID:        userID,
		SubmittedFlag: req.Flag,
		IPAddress:     c.ClientIP(),
	}

	isCorrect := false
	var dynamicContainer models.Container
	if challenge.Mode == models.ChallengeModeStatic {
		isCorrect = (challenge.StaticFlag == req.Flag)
	} else {
		err := database.DB.Where("challenge_id = ? AND team_id = ?", challengeID, userTeam.TeamID).First(&dynamicContainer).Error
		if err == nil && dynamicContainer.ContainerFlag == req.Flag {
			isCorrect = true
		}
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var existingSolve models.Submission
		if err := tx.Where("challenge_id = ? AND team_id = ?", challengeID, userTeam.TeamID).First(&existingSolve).Error; err == nil {
			logEntry.FlagResult = models.FlagResultDuplicate
			tx.Create(&logEntry)
			utils.Error(c, 6001, "Already solved by your team")
			return errors.New("duplicate solve")
		}

		if !isCorrect {
			logEntry.FlagResult = models.FlagResultWrong
			tx.Create(&logEntry)
			utils.Error(c, 6002, "Incorrect flag")
			return errors.New("incorrect flag")
		}

		logEntry.FlagResult = models.FlagResultCorrect
		if err := tx.Create(&logEntry).Error; err != nil {
			return err
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&challenge, challengeID).Error; err != nil {
			return err
		}

		scoreToAward := challenge.CurrentScore
		newSolve := models.Submission{
			ChallengeID: uint32(challengeID),
			TeamID:      userTeam.TeamID,
			UserID:      userID,
			Score:       scoreToAward,
		}
		if err := tx.Create(&newSolve).Error; err != nil {
			return err
		}

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

		if err := tx.Save(&challenge).Error; err != nil {
			return err
		}

		if isCorrect && challenge.Mode == models.ChallengeModeDynamic && dynamicContainer.ID != 0 {
			go func() {
				err := services.DestroyService(dynamicContainer.DockerID) // 使用 DestroyService
				if err != nil {
					log.Printf("Error destroying container %s after solve: %v", dynamicContainer.DockerID, err)
					return
				}
				dynamicContainer.State = models.ContainerStateDestroyed
				database.DB.Save(&dynamicContainer)
				log.Printf("Container %s destroyed successfully after correct submission.", dynamicContainer.DockerID)
			}()
		}

		if challenge.Mode == models.ChallengeModeDynamic {
			go func(flag string, currentTeamID uint32) {
				var otherSubmissions []models.SubmissionLog
				database.DB.Where("submitted_flag = ? AND team_id != ? AND flag_result = ?", flag, currentTeamID, models.FlagResultCorrect).Find(&otherSubmissions)
				if len(otherSubmissions) > 0 {
					database.DB.Model(&models.SubmissionLog{}).Where("submitted_flag = ? AND flag_result = ?", flag, models.FlagResultCorrect).Update("suspected", true)
					log.Printf("Suspicious activity detected: Dynamic flag '%s' submitted by multiple teams. All related submissions have been marked.", flag)
				}
			}(req.Flag, team.ID)
		}

		// 新增：触发大屏缓存更新
		go func(solve models.Submission, chal models.Challenge, t models.Team) {
			services.AddSolveToFeed(solve, chal, t)
			services.UpdateScoreboardCache() // 每次解题都完全刷新排行榜
		}(newSolve, challenge, team)

		utils.Success(c, "Correct! First solve for your team.", gin.H{
			"challenge_id": challenge.ID,
			"score":        scoreToAward,
			"team_id":      team.ID,
			"solving_time": newSolve.SolvingTime,
		})
		return nil
	})

	if err != nil && (err.Error() == "duplicate solve" || err.Error() == "incorrect flag") {
		return
	}

	// 提交Flag后（无论成功失败），清理该题目的详情缓存，以保证分数和解题数能及时刷新
	cacheKey := "challenge_detail:" + strconv.Itoa(challengeID)
	database.RDB.Del(database.Ctx, cacheKey)
}

// UpdateChallenge —— 管理员修改题目
func UpdateChallenge(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的题目ID")
		return
	}

	// 在执行更新前，先删除缓存
	cacheKey := "challenge_detail:" + strconv.Itoa(id)
	database.RDB.Del(database.Ctx, cacheKey)

	var req dto.UpdateChallengeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	var challenge models.Challenge
	if err := database.DB.First(&challenge, id).Error; err != nil {
		utils.Error(c, 4004, "题目不存在")
		return
	}

	updates := make(map[string]interface{})
	if req.State != nil {
		updates["state"] = models.ChallengeState(*req.State)
	}
	if req.Hint != nil {
		updates["hint"] = *req.Hint
	}
	if req.Difficulty != nil {
		updates["difficulty"] = models.ChallengeDifficulty(*req.Difficulty)
	}
	if req.Mode != nil {
		updates["mode"] = models.ChallengeMode(*req.Mode)
	}
	if req.StaticFlag != nil {
		updates["static_flag"] = *req.StaticFlag
	}
	if req.DockerImage != nil {
		updates["docker_image"] = *req.DockerImage
	}
	if req.DockerPorts != nil {
		updates["docker_ports"] = *req.DockerPorts
	}

	if len(updates) == 0 {
		utils.Success(c, "没有需要更新的字段", nil)
		return
	}

	if err := database.DB.Model(&challenge).Updates(updates).Error; err != nil {
		utils.Error(c, 5000, "更新题目失败: "+err.Error())
		return
	}

	utils.Success(c, "Challenge updated successfully", nil)
}

// DeleteChallenge —— 管理员删除题目
func DeleteChallenge(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的题目ID")
		return
	}

	if err := database.DB.Delete(&models.Challenge{}, id).Error; err != nil {
		utils.Error(c, 5000, "删除题目失败: "+err.Error())
		return
	}

	utils.Success(c, "Challenge deleted successfully", nil)
}

// AdminListChallenges —— 管理员查询题目列表
func AdminListChallenges(c *gin.Context) {
	typeIDStr := c.Query("type_id")
	mode := strings.TrimSpace(c.Query("mode"))
	diff := strings.TrimSpace(c.Query("difficulty"))
	state := strings.TrimSpace(c.Query("state"))
	kw := strings.TrimSpace(c.Query("keyword"))
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

// AdminGetChallengeDetail —— 管理员查询题目详情
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
