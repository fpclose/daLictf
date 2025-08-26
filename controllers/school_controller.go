// file: controllers/school_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"github.com/gin-gonic/gin"
	"strconv"
)

func CreateSchool(c *gin.Context) {
	var req struct {
		SchoolName string `json:"school_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	var school models.School
	if err := database.DB.Where("school_name = ?", req.SchoolName).First(&school).Error; err == nil {
		utils.Error(c, 2001, "School already exists")
		return
	}

	// 生成唯一的邀请码
	var invitationCode string
	for {
		invitationCode = utils.GenerateInvitationCode(8) // 生成8位邀请码
		var count int64
		database.DB.Model(&models.School{}).Where("invitation_code = ?", invitationCode).Count(&count)
		if count == 0 {
			break
		}
	}

	newSchool := models.School{
		SchoolName:     req.SchoolName,
		InvitationCode: invitationCode,
	}

	if err := database.DB.Create(&newSchool).Error; err != nil {
		utils.Error(c, 5000, "数据库错误: "+err.Error())
		return
	}

	utils.Success(c, "School created successfully", models.AdminSchoolInfo{
		ID:             newSchool.ID,
		SchoolName:     newSchool.SchoolName,
		InvitationCode: newSchool.InvitationCode,
		UserCount:      newSchool.UserCount,
		Status:         newSchool.Status,
	})
}

func GetSchoolList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "id")
	order := c.DefaultQuery("order", "desc")
	status := c.Query("status")

	var schools []models.School
	var total int64

	db := database.DB.Model(&models.School{})

	if search != "" {
		db = db.Where("school_name LIKE ?", "%"+search+"%")
	}

	// 权限控制：只有管理员才能按 status 筛选
	role, _ := c.Get("user_role")
	if role == models.RoleAdmin || role == models.RoleRootAdmin {
		if status != "" {
			db = db.Where("status = ?", status)
		}
	} else {
		// 普通用户只能看到 active 的学校
		db = db.Where("status = ?", models.SchoolStatusActive)
	}

	db.Count(&total)
	db.Order(sortBy + " " + order).Offset((page - 1) * limit).Limit(limit).Find(&schools)

	publicSchools := make([]models.PublicSchoolInfo, len(schools))
	for i, school := range schools {
		publicSchools[i] = models.PublicSchoolInfo{
			ID:         school.ID,
			SchoolName: school.SchoolName,
			UserCount:  school.UserCount,
		}
	}

	utils.Success(c, "success", gin.H{
		"total":   total,
		"schools": publicSchools,
	})
}

func GetSchoolDetail(c *gin.Context) {
	schoolID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的学校ID")
		return
	}

	var school models.School
	if err := database.DB.First(&school, schoolID).Error; err != nil {
		utils.Error(c, 4004, "学校不存在")
		return
	}

	role, _ := c.Get("user_role")
	if role == models.RoleAdmin || role == models.RoleRootAdmin {
		utils.Success(c, "success", models.AdminSchoolInfo{
			ID:             school.ID,
			SchoolName:     school.SchoolName,
			InvitationCode: school.InvitationCode,
			UserCount:      school.UserCount,
			Status:         school.Status,
			CreatedAt:      school.CreatedAt,
			UpdatedAt:      school.UpdatedAt,
		})
	} else {
		utils.Success(c, "success", gin.H{
			"id":          school.ID,
			"school_name": school.SchoolName,
			"user_count":  school.UserCount,
			"status":      school.Status,
			"created_at":  school.CreatedAt,
		})
	}
}

func UpdateSchool(c *gin.Context) {
	schoolID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		SchoolName string `json:"school_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	// 检查新名称是否与其它学校冲突
	var existingSchool models.School
	if err := database.DB.Where("school_name = ? AND id != ?", req.SchoolName, schoolID).First(&existingSchool).Error; err == nil {
		utils.Error(c, 2001, "School name already exists")
		return
	}

	var schoolToUpdate models.School
	if err := database.DB.First(&schoolToUpdate, schoolID).Error; err != nil {
		utils.Error(c, 4004, "学校不存在")
		return
	}

	schoolToUpdate.SchoolName = req.SchoolName
	database.DB.Save(&schoolToUpdate)

	utils.Success(c, "School updated successfully", nil)
}

func DeleteSchool(c *gin.Context) {
	schoolID, _ := strconv.Atoi(c.Param("id"))
	// GORM软删除会自动处理
	if err := database.DB.Delete(&models.School{}, schoolID).Error; err != nil {
		utils.Error(c, 5000, "数据库错误")
		return
	}
	utils.Success(c, "School deleted successfully", nil)
}

func UpdateSchoolStatus(c *gin.Context) {
	schoolID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status models.SchoolStatus `json:"status" binding:"required,oneof=active suspended"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "无效的状态值")
		return
	}

	var school models.School
	if err := database.DB.First(&school, schoolID).Error; err != nil {
		utils.Error(c, 4004, "学校不存在")
		return
	}

	database.DB.Model(&school).Update("status", req.Status)
	utils.Success(c, "School status updated successfully", gin.H{
		"id":     school.ID,
		"status": req.Status,
	})
}

func ResetInvitationCode(c *gin.Context) {
	schoolID, _ := strconv.Atoi(c.Param("id"))
	var school models.School
	if err := database.DB.First(&school, schoolID).Error; err != nil {
		utils.Error(c, 4004, "学校不存在")
		return
	}

	// 生成唯一的邀请码
	var newInvitationCode string
	for {
		newInvitationCode = utils.GenerateInvitationCode(8)
		var count int64
		database.DB.Model(&models.School{}).Where("invitation_code = ?", newInvitationCode).Count(&count)
		if count == 0 {
			break
		}
	}

	database.DB.Model(&school).Update("invitation_code", newInvitationCode)
	utils.Success(c, "Invitation code reset successfully", gin.H{
		"id":                  school.ID,
		"new_invitation_code": newInvitationCode,
	})
}
