// file: controllers/attachment_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/dto"
	"ISCTF/models"
	"ISCTF/utils"
	"crypto/sha256"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// AddAttachment —— 支持 JSON 外链 & multipart 上传，使用 DTO 绑定
func AddAttachment(c *gin.Context) {
	challengeID, _ := strconv.Atoi(c.Param("id"))

	// 从中间件获取用户 ID
	userIDAny, ok := c.Get("user_id")
	if !ok {
		utils.Error(c, 4001, "未登录")
		return
	}
	userID := userIDAny.(uint32)

	contentType := c.ContentType()
	var newAttachment models.Attachment
	newAttachment.ChallengeID = uint32(challengeID)
	newAttachment.CreatedBy = userID

	if contentType == "application/json" {
		// 外链方式
		var req dto.AddAttachmentURLReq
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.Error(c, 1001, "参数无效")
			return
		}
		newAttachment.Storage = models.StorageURL
		newAttachment.URL = req.URL
		newAttachment.FileName = req.FileName
		newAttachment.SHA256 = "URL_NOT_HASHED"
		newAttachment.Status = models.AttachmentStatusActive // 外链直接激活（若需风控可改为 pending_scan）

	} else if strings.HasPrefix(contentType, "multipart/") {
		// 平台上传方式
		file, err := c.FormFile("file")
		if err != nil {
			utils.Error(c, 1001, "获取文件失败")
			return
		}

		// 保存到本地（示例路径：./uploads）
		if err := os.MkdirAll("./uploads", 0o755); err != nil {
			utils.Error(c, 5000, "创建上传目录失败")
			return
		}
		dst := filepath.Join("./uploads", file.Filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			utils.Error(c, 5000, "保存文件失败")
			return
		}

		// 计算 SHA256
		f, err := os.Open(dst)
		if err != nil {
			utils.Error(c, 5000, "打开文件失败")
			return
		}
		defer f.Close()

		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			utils.Error(c, 5000, "计算哈希失败")
			return
		}

		newAttachment.Storage = models.StorageObject
		newAttachment.ObjectBucket = "" // 如使用对象存储可设置真实桶名
		newAttachment.ObjectKey = dst   // 示例：本地文件路径
		newAttachment.FileName = file.Filename
		newAttachment.ContentType = file.Header.Get("Content-Type")
		newAttachment.FileSize = uint64(file.Size)
		newAttachment.SHA256 = hex.EncodeToString(hasher.Sum(nil))
		newAttachment.Status = models.AttachmentStatusPendingScan // 默认待扫描，后续可异步转 active

	} else {
		utils.Error(c, 1001, "不支持的 Content-Type")
		return
	}

	if err := database.DB.Create(&newAttachment).Error; err != nil {
		utils.Error(c, 5000, "创建附件记录失败")
		return
	}

	utils.Success(c, "success", gin.H{
		"attachment_id": newAttachment.ID,
		"status":        newAttachment.Status,
	})
}

// DownloadAttachment —— 统一网关下载：外链 302，本地文件直接返回
func DownloadAttachment(c *gin.Context) {
	attachmentID, _ := strconv.Atoi(c.Param("attachment_id"))

	var attachment models.Attachment
	if err := database.DB.First(&attachment, attachmentID).Error; err != nil {
		utils.Error(c, 4004, "附件不存在")
		return
	}

	if attachment.Storage == models.StorageURL {
		c.Redirect(302, attachment.URL)
		return
	}

	// 本地/对象存储代理下载（示例直接读 ObjectKey）
	if attachment.ObjectKey == "" {
		utils.Error(c, 5000, "对象存储路径为空")
		return
	}
	c.File(attachment.ObjectKey)
}

// ListAttachments —— 列出题目所有附件
func ListAttachments(c *gin.Context) {
	challengeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, 1002, "无效的题目ID")
		return
	}

	var attachments []models.Attachment
	db := database.DB.Where("challenge_id = ?", challengeID)

	// 非管理员用户只能看到 active 状态的附件
	role, _ := c.Get("user_role")
	if role != models.RoleAdmin && role != models.RoleRootAdmin {
		db = db.Where("status = ?", models.AttachmentStatusActive)
	}

	if err := db.Find(&attachments).Error; err != nil {
		utils.Error(c, 5000, "查询附件失败: "+err.Error())
		return
	}

	utils.Success(c, "success", attachments)
}

// UpdateAttachmentStatus —— 管理员更新附件状态
func UpdateAttachmentStatus(c *gin.Context) {
	attachmentID, err := strconv.Atoi(c.Param("attachment_id"))
	if err != nil {
		utils.Error(c, 1002, "无效的附件ID")
		return
	}

	var req struct {
		Status models.AttachmentStatus `json:"status" binding:"required,oneof=pending_scan active quarantined archived"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 1001, "参数无效: "+err.Error())
		return
	}

	var attachment models.Attachment
	if err := database.DB.First(&attachment, attachmentID).Error; err != nil {
		utils.Error(c, 4004, "附件不存在")
		return
	}

	if err := database.DB.Model(&attachment).Update("status", req.Status).Error; err != nil {
		utils.Error(c, 5000, "更新附件状态失败: "+err.Error())
		return
	}

	utils.Success(c, "Attachment status updated successfully", nil)
}

// DeleteAttachment —— 管理员删除附件
func DeleteAttachment(c *gin.Context) {
	attachmentID, err := strconv.Atoi(c.Param("attachment_id"))
	if err != nil {
		utils.Error(c, 1002, "无效的附件ID")
		return
	}

	var attachment models.Attachment
	if err := database.DB.First(&attachment, attachmentID).Error; err != nil {
		// 如果记录不存在，也认为是成功的删除操作
		utils.Success(c, "Attachment deleted successfully", nil)
		return
	}

	// 如果是对象存储（本地文件），则尝试删除本地文件
	if attachment.Storage == models.StorageObject && attachment.ObjectKey != "" {
		// 注意：生产环境应增加更严格的路径校验，防止路径遍历
		_ = os.Remove(attachment.ObjectKey)
	}

	if err := database.DB.Delete(&attachment).Error; err != nil {
		utils.Error(c, 5000, "删除附件记录失败: "+err.Error())
		return
	}

	utils.Success(c, "Attachment deleted successfully", nil)
}

// RescanAttachment —— 重新扫描附件（占位符）
func RescanAttachment(c *gin.Context) {
	_, err := strconv.Atoi(c.Param("attachment_id"))
	if err != nil {
		utils.Error(c, 1002, "无效的附件ID")
		return
	}

	// 此处为功能占位，实际应触发异步扫描任务
	utils.Success(c, "Rescan scheduled", nil)
}
