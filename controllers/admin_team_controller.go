// file: controllers/admin_team_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"github.com/gin-gonic/gin"
	"strconv"
)

func AdminGetTeams(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")

	var teams []models.Team
	var total int64

	db := database.DB.Model(&models.Team{}).Preload("Leader").Preload("School")

	if search != "" {
		db = db.Where("team_name LIKE ?", "%"+search+"%")
	}

	db.Count(&total)
	db.Order("id desc").Offset((page - 1) * limit).Limit(limit).Find(&teams)

	// 为了返回更清晰的数据，我们创建一个自定义的结构
	type TeamInfo struct {
		ID             uint32            `json:"id"`
		TeamName       string            `json:"team_name"`
		LeaderUsername string            `json:"leader_username"`
		SchoolName     *string           `json:"school_name"`
		Track          models.UserTrack  `json:"track"`
		TeamStatus     models.TeamStatus `json:"team_status"`
		MemberCount    int64             `json:"member_count"`
	}

	var resultTeams []TeamInfo
	for _, team := range teams {
		var schoolName *string
		if team.School != nil {
			schoolName = &team.School.SchoolName
		}
		var memberCount int64
		database.DB.Model(&models.TeamMember{}).Where("team_id = ?", team.ID).Count(&memberCount)

		resultTeams = append(resultTeams, TeamInfo{
			ID:             team.ID,
			TeamName:       team.TeamName,
			LeaderUsername: team.Leader.Username,
			SchoolName:     schoolName,
			Track:          team.Track,
			TeamStatus:     team.TeamStatus,
			MemberCount:    memberCount,
		})
	}

	utils.Success(c, "success", gin.H{
		"total": total,
		"teams": resultTeams,
	})
}

func AdminUpdateTeamStatus(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的队伍ID")
		return
	}

	var req struct {
		Status models.TeamStatus `json:"status" binding:"required,oneof=active banned hidden"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "无效的状态值")
		return
	}

	var team models.Team
	if err := database.DB.First(&team, teamID).Error; err != nil {
		utils.Error(c, 4004, "队伍不存在")
		return
	}

	if err := database.DB.Model(&team).Update("team_status", req.Status).Error; err != nil {
		utils.Error(c, 5000, "更新队伍状态失败")
		return
	}

	utils.Success(c, "Team status updated successfully", gin.H{
		"team_id": team.ID,
		"status":  req.Status,
	})
}

func AdminDeleteTeam(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的队伍ID")
		return
	}

	// 管理员删除是硬删除，GORM的级联删除会处理成员关系
	if err := database.DB.Select("Members").Delete(&models.Team{ID: uint32(teamID)}).Error; err != nil {
		utils.Error(c, 5000, "删除队伍失败")
		return
	}

	utils.Success(c, "Team deleted successfully by admin", nil)
}
