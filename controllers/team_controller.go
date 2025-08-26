// file: controllers/team_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strconv"
	"time" // 确保导入了 time 包
)

// isUserInTeam 是一个辅助函数，检查用户是否已在队伍中
func isUserInTeam(userID uint32) (bool, error) {
	var count int64
	err := database.DB.Model(&models.TeamMember{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func CreateTeam(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint32)

	inTeam, err := isUserInTeam(userID)
	if err != nil {
		utils.Error(c, 5000, "数据库错误")
		return
	}
	if inTeam {
		utils.Error(c, 3001, "User already in a team")
		return
	}

	var req struct {
		TeamName     string `json:"team_name" binding:"required"`
		TeamDescribe string `json:"team_describe"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效")
		return
	}

	var existingTeam models.Team
	if err := database.DB.Where("team_name = ?", req.TeamName).First(&existingTeam).Error; err == nil {
		utils.Error(c, 3001, "Team name already exists")
		return
	}

	var currentUser models.User
	database.DB.First(&currentUser, userID)

	invitationCode := utils.GenerateInvitationCode(12)

	newTeam := models.Team{
		TeamName:       req.TeamName,
		LeaderID:       userID,
		SchoolID:       currentUser.SchoolID,
		Track:          currentUser.Track,
		InvitationCode: invitationCode,
		TeamDescribe:   req.TeamDescribe,
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newTeam).Error; err != nil {
			return err
		}
		leaderMember := models.TeamMember{
			TeamID: newTeam.ID,
			UserID: userID,
			Role:   models.TeamRoleLeader,
			// =====================================
			//  ↓↓↓ 新增：显式设置加入时间 ↓↓↓
			// =====================================
			JoinedAt: time.Now(),
		}
		if err := tx.Create(&leaderMember).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		utils.Error(c, 5000, "创建队伍失败: "+err.Error())
		return
	}

	utils.Success(c, "Team created successfully", gin.H{
		"id":              newTeam.ID,
		"team_name":       newTeam.TeamName,
		"leader_id":       newTeam.LeaderID,
		"school_id":       newTeam.SchoolID,
		"track":           newTeam.Track,
		"invitation_code": newTeam.InvitationCode,
	})
}

func JoinTeam(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint32)

	var req struct {
		InvitationCode string `json:"invitation_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效")
		return
	}

	inTeam, err := isUserInTeam(userID)
	if err != nil {
		utils.Error(c, 5000, "数据库错误")
		return
	}
	if inTeam {
		utils.Error(c, 3001, "User already in a team")
		return
	}

	var targetTeam models.Team
	if err := database.DB.Where("invitation_code = ?", req.InvitationCode).First(&targetTeam).Error; err != nil {
		utils.Error(c, 3004, "Invalid invitation code")
		return
	}

	var currentUser models.User
	database.DB.First(&currentUser, userID)

	if currentUser.Track != targetTeam.Track {
		utils.Error(c, 3002, "Track conflict")
		return
	}
	if (currentUser.SchoolID != nil && targetTeam.SchoolID != nil && *currentUser.SchoolID != *targetTeam.SchoolID) || (currentUser.SchoolID == nil && targetTeam.SchoolID != nil) || (currentUser.SchoolID != nil && targetTeam.SchoolID == nil) {
		utils.Error(c, 3003, "School conflict")
		return
	}

	newMember := models.TeamMember{
		TeamID: targetTeam.ID,
		UserID: userID,
		Role:   models.TeamRoleMember,
		// =====================================
		//  ↓↓↓ 新增：显式设置加入时间 ↓↓↓
		// =====================================
		JoinedAt: time.Now(),
	}
	if err := database.DB.Create(&newMember).Error; err != nil {
		utils.Error(c, 5000, "加入队伍失败")
		return
	}

	utils.Success(c, "Joined team successfully", gin.H{
		"team_id":   targetTeam.ID,
		"team_name": targetTeam.TeamName,
	})
}

func LeaveTeam(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint32)

	var member models.TeamMember
	if err := database.DB.Where("user_id = ?", userID).First(&member).Error; err != nil {
		utils.Error(c, 3005, "User not in any team")
		return
	}

	if member.Role == models.TeamRoleLeader {
		utils.Error(c, 3006, "Leader cannot leave team, please transfer leadership or disband the team")
		return
	}

	if err := database.DB.Delete(&member).Error; err != nil {
		utils.Error(c, 5000, "退出队伍失败")
		return
	}

	utils.Success(c, "Left team successfully", nil)
}

func KickMember(c *gin.Context) {
	teamID, _ := strconv.Atoi(c.Param("id"))
	memberUserID, _ := strconv.Atoi(c.Param("user_id"))

	leaderIDAny, _ := c.Get("user_id")
	leaderID := leaderIDAny.(uint32)

	var team models.Team
	if err := database.DB.First(&team, teamID).Error; err != nil || team.LeaderID != leaderID {
		utils.Error(c, 4003, "Permission denied: not the team leader")
		return
	}

	if uint32(memberUserID) == leaderID {
		utils.Error(c, 3008, "Cannot kick the leader")
		return
	}

	result := database.DB.Where("team_id = ? AND user_id = ?", teamID, memberUserID).Delete(&models.TeamMember{})
	if result.Error != nil {
		utils.Error(c, 5000, "移除队员失败")
		return
	}
	if result.RowsAffected == 0 {
		utils.Error(c, 3007, "Member not found in this team")
		return
	}

	utils.Success(c, "Member removed successfully", nil)
}

func DisbandTeam(c *gin.Context) {
	teamID, _ := strconv.Atoi(c.Param("id"))

	leaderIDAny, _ := c.Get("user_id")
	leaderID := leaderIDAny.(uint32)

	var team models.Team
	if err := database.DB.First(&team, teamID).Error; err != nil {
		utils.Error(c, 4004, "Team not found")
		return
	}

	if team.LeaderID != leaderID {
		utils.Error(c, 4003, "Permission denied: not the team leader")
		return
	}

	if err := database.DB.Delete(&team).Error; err != nil {
		utils.Error(c, 5000, "解散队伍失败")
		return
	}

	utils.Success(c, "Team disbanded successfully", nil)
}

func UpdateTeam(c *gin.Context) {
	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的队伍ID")
		return
	}
	leaderIDAny, _ := c.Get("user_id")
	leaderID := leaderIDAny.(uint32)

	var team models.Team
	if err := database.DB.First(&team, teamID).Error; err != nil {
		utils.Error(c, 4004, "队伍不存在")
		return
	}

	if team.LeaderID != leaderID {
		utils.Error(c, 4003, "权限不足，只有队长可以修改队伍信息")
		return
	}

	var req struct {
		TeamDescribe string `json:"team_describe"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效")
		return
	}

	if err := database.DB.Model(&team).Update("team_describe", req.TeamDescribe).Error; err != nil {
		utils.Error(c, 5000, "更新队伍信息失败")
		return
	}

	utils.Success(c, "Team updated successfully", nil)
}
