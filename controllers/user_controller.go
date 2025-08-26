// file: controllers/user_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"github.com/gin-gonic/gin"
	"strconv"
)

// --- 公开接口 ---

func Register(c *gin.Context) {
	var req struct {
		Username             string `json:"username" binding:"required"`
		Password             string `json:"password" binding:"required,min=8"`
		Email                string `json:"email" binding:"required,email"`
		RealName             string `json:"real_name"`
		SchoolInvitationCode string `json:"school_invitation_code"`
		StudentNumber        string `json:"student_number"`
		GradeYear            *int   `json:"grade_year"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	var user models.User
	if err := database.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&user).Error; err == nil {
		utils.Error(c, 2001, "用户名或邮箱已被注册")
		return
	}

	newUser := models.User{
		Username:      req.Username,
		Password:      req.Password,
		Email:         req.Email,
		RealName:      req.RealName,
		StudentNumber: req.StudentNumber,
		GradeYear:     req.GradeYear,
	}

	if req.SchoolInvitationCode != "" {
		var school models.School
		if err := database.DB.Where("invitation_code = ?", req.SchoolInvitationCode).First(&school).Error; err != nil {
			utils.Error(c, 2003, "Invalid school invitation code")
			return
		}

		// 新增：校验学校状态是否为 active
		if school.Status != models.SchoolStatusActive {
			utils.Error(c, 2003, "Invalid school invitation code") // 同样返回此错误，不暴露学校具体状态
			return
		}

		newUser.SchoolID = &school.ID
	}

	contestStartTimeYear := 2025 // 假设比赛在2025年开始
	if newUser.SchoolID == nil {
		newUser.Track = models.TrackSociety
	} else if req.GradeYear != nil {
		if *req.GradeYear == contestStartTimeYear {
			newUser.Track = models.TrackFreshman
		} else if *req.GradeYear < contestStartTimeYear {
			newUser.Track = models.TrackAdvanced
		}
	} else {
		newUser.Track = models.TrackSociety
	}

	if err := database.DB.Create(&newUser).Error; err != nil {
		utils.Error(c, 5000, "数据库错误: "+err.Error())
		return
	}

	utils.Success(c, "User registered successfully", gin.H{
		"id":         newUser.ID,
		"username":   newUser.Username,
		"grade_year": newUser.GradeYear,
		"track":      newUser.Track,
		"role":       newUser.Role,
	})
}

func Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		utils.Error(c, 2002, "用户不存在或密码错误")
		return
	}

	if !user.CheckPassword(req.Password) {
		utils.Error(c, 2002, "用户不存在或密码错误")
		return
	}

	if user.Status == models.StatusBanned {
		utils.Error(c, 2005, "用户已被封禁")
		return
	}

	token, err := utils.GenerateToken(user)
	if err != nil {
		utils.Error(c, 5002, "Token 生成失败")
		return
	}

	utils.Success(c, "Login success", gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"track":    user.Track,
			"role":     user.Role,
		},
	})
}

// --- 需要登录的接口 ---

func GetUserDetail(c *gin.Context) {
	targetUserID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的用户ID")
		return
	}
	requestingUserID := c.GetUint("user_id")
	requestingUserRole := c.GetString("user_role")
	if uint(targetUserID) != requestingUserID && requestingUserRole == string(models.RoleUser) {
		utils.Error(c, 4003, "权限不足")
		return
	}
	var user models.User
	if err := database.DB.Preload("School").First(&user, targetUserID).Error; err != nil {
		utils.Error(c, 4004, "用户不存在")
		return
	}
	var schoolName *string
	if user.School != nil {
		schoolName = &user.School.SchoolName
	}
	utils.Success(c, "success", gin.H{
		"id":             user.ID,
		"username":       user.Username,
		"real_name":      user.RealName,
		"school_name":    schoolName,
		"student_number": user.StudentNumber,
		"grade_year":     user.GradeYear,
		"track":          user.Track,
		"role":           user.Role,
	})
}

func UpdateUser(c *gin.Context) {
	utils.Success(c, "用户信息更新成功（待实现）", nil)
}

// --- 仅管理员可访问的接口 ---

func GetUserList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	query := c.Query("query")
	var users []models.User
	var total int64
	db := database.DB.Model(&models.User{}).Preload("School")
	if query != "" {
		db = db.Where("username LIKE ? OR real_name LIKE ? OR email LIKE ?", "%"+query+"%", "%"+query+"%", "%"+query+"%")
	}
	db.Count(&total)
	db.Offset((page - 1) * pageSize).Limit(pageSize).Order("id desc").Find(&users)
	var resultUsers []gin.H
	for _, user := range users {
		var schoolName *string
		if user.School != nil {
			schoolName = &user.School.SchoolName
		}
		resultUsers = append(resultUsers, gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"school_name": schoolName,
			"grade_year":  user.GradeYear,
			"track":       user.Track,
			"role":        user.Role,
			"status":      user.Status,
		})
	}
	utils.Success(c, "success", gin.H{
		"total": total,
		"users": resultUsers,
	})
}

func DeleteUser(c *gin.Context) {
	targetUserID, _ := strconv.Atoi(c.Param("id"))
	var user models.User
	if err := database.DB.First(&user, targetUserID).Error; err != nil {
		utils.Error(c, 4004, "用户不存在")
		return
	}
	if user.Role == models.RoleRootAdmin {
		utils.Error(c, 2011, "Root admin cannot be deleted")
		return
	}
	database.DB.Delete(&user)
	utils.Success(c, "User deleted successfully", nil)
}

func UpdateUserStatus(c *gin.Context) {
	targetUserID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status models.UserStatus `json:"status" binding:"required,oneof=active banned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "无效的状态")
		return
	}
	var user models.User
	if err := database.DB.First(&user, targetUserID).Error; err != nil {
		utils.Error(c, 4004, "用户不存在")
		return
	}
	if user.Role == models.RoleRootAdmin {
		utils.Error(c, 2010, "Root admin cannot be modified")
		return
	}
	database.DB.Model(&user).Update("status", req.Status)
	utils.Success(c, "User status updated", gin.H{
		"user_id": user.ID,
		"status":  req.Status,
	})
}

// --- 仅 Root Admin 可访问的接口 ---

func UpdateUserRole(c *gin.Context) {
	targetUserID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Role models.UserRole `json:"role" binding:"required,oneof=user admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "无效的角色")
		return
	}
	var user models.User
	if err := database.DB.First(&user, targetUserID).Error; err != nil {
		utils.Error(c, 4004, "用户不存在")
		return
	}
	if user.Role == models.RoleRootAdmin {
		utils.Error(c, 2010, "Root admin cannot be modified")
		return
	}
	database.DB.Model(&user).Update("role", req.Role)
	utils.Success(c, "Role updated successfully", gin.H{
		"user_id": user.ID,
		"role":    req.Role,
	})
}
