// file: controllers/question_type_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"github.com/gin-gonic/gin"
	"strconv"
)

// CreateQuestionType 新增题目类型
func CreateQuestionType(c *gin.Context) {
	var req struct {
		Direction   string `json:"direction" binding:"required"`
		Alias       string `json:"alias"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	var existingType models.QuestionType
	if err := database.DB.Where("direction = ?", req.Direction).First(&existingType).Error; err == nil {
		utils.Error(c, 4001, "Question type already exists")
		return
	}

	newType := models.QuestionType{
		Direction:   req.Direction,
		Alias:       req.Alias,
		Description: req.Description,
	}

	if err := database.DB.Create(&newType).Error; err != nil {
		utils.Error(c, 5000, "数据库错误: "+err.Error())
		return
	}

	utils.Success(c, "Question type created successfully", gin.H{
		"id":        newType.ID,
		"direction": newType.Direction,
	})
}

// GetQuestionTypeList 查询题目类型列表
func GetQuestionTypeList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")

	var types []models.QuestionType
	var total int64

	db := database.DB.Model(&models.QuestionType{})

	if search != "" {
		db = db.Where("direction LIKE ? OR alias LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	db.Count(&total)
	db.Order("id asc").Offset((page - 1) * limit).Limit(limit).Find(&types)

	utils.Success(c, "success", gin.H{
		"total": total,
		"types": types,
	})
}

// GetQuestionTypeDetail 查询单个题目类型详情
func GetQuestionTypeDetail(c *gin.Context) {
	typeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的ID")
		return
	}

	var questionType models.QuestionType
	if err := database.DB.First(&questionType, typeID).Error; err != nil {
		utils.Error(c, 4004, "题目类型不存在")
		return
	}

	utils.Success(c, "success", questionType)
}

// UpdateQuestionType 修改题目类型
func UpdateQuestionType(c *gin.Context) {
	typeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的ID")
		return
	}

	var req struct {
		Alias       string `json:"alias"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	var questionType models.QuestionType
	if err := database.DB.First(&questionType, typeID).Error; err != nil {
		utils.Error(c, 4004, "题目类型不存在")
		return
	}

	// 更新字段
	questionType.Alias = req.Alias
	questionType.Description = req.Description

	if err := database.DB.Save(&questionType).Error; err != nil {
		utils.Error(c, 5000, "更新失败: "+err.Error())
		return
	}

	utils.Success(c, "Question type updated successfully", nil)
}

// DeleteQuestionType 删除题目类型
func DeleteQuestionType(c *gin.Context) {
	typeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的ID")
		return
	}

	// 业务逻辑检查：删除前确认没有题目绑定此类型
	// 注意：这里我们假设未来会有一个 Challenge 模型，它有一个 TypeID 字段。
	// 现在我们用一个 map 来模拟这个模型进行查询。
	var challengeCount int64
	// GORM 可以直接使用表名进行查询
	database.DB.Table("dalictf_challenge").Where("type_id = ?", typeID).Count(&challengeCount)

	if challengeCount > 0 {
		utils.Error(c, 4002, "Cannot delete type with existing challenges")
		return
	}

	if err := database.DB.Delete(&models.QuestionType{}, typeID).Error; err != nil {
		utils.Error(c, 5000, "删除失败: "+err.Error())
		return
	}

	utils.Success(c, "Question type deleted successfully", nil)
}
