// file: controllers/attachment_controller.go
package controllers

import (
	"ISCTF/database"
	"ISCTF/models"
	"ISCTF/utils"
	"crypto/sha256"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

func AddAttachment(c *gin.Context) {
	challengeID, _ := strconv.Atoi(c.Param("id"))

	// =======================================================
	//  ↓↓↓ 修正点：使用兼容旧版Gin的写法 ↓↓↓
	// =======================================================
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint32)
	// =======================================================

	contentType := c.ContentType()

	var newAttachment models.Attachment
	newAttachment.ChallengeID = uint32(challengeID)
	newAttachment.CreatedBy = userID

	if contentType == "application/json" {
		var req struct {
			URL      string `json:"url" binding:"required,url"`
			FileName string `json:"file_name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.Error(c, 1001, "参数无效")
			return
		}
		newAttachment.Storage = models.StorageURL
		newAttachment.URL = req.URL
		newAttachment.FileName = req.FileName
		newAttachment.SHA256 = "URL_NOT_HASHED"
		// =======================================================
		//  ↓↓↓ 修正点：使用带前缀的新常量 ↓↓↓
		// =======================================================
		newAttachment.Status = models.AttachmentStatusActive
		// =======================================================
	} else if c.Request.Header.Get("Content-Type")[0:19] == "multipart/form-data" { // 更稳健地判断
		file, err := c.FormFile("file")
		if err != nil {
			utils.Error(c, 1001, "获取文件失败")
			return
		}

		dst := filepath.Join("./uploads", file.Filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			utils.Error(c, 5000, "保存文件失败")
			return
		}

		f, _ := os.Open(dst)
		defer f.Close()
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			utils.Error(c, 5000, "计算哈希失败")
			return
		}

		newAttachment.Storage = models.StorageObject
		newAttachment.ObjectKey = dst
		newAttachment.FileName = file.Filename
		newAttachment.FileSize = uint64(file.Size)
		newAttachment.ContentType = file.Header.Get("Content-Type")
		newAttachment.SHA256 = hex.EncodeToString(hasher.Sum(nil))
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

func DownloadAttachment(c *gin.Context) {
	attachmentID, _ := strconv.Atoi(c.Param("attachment_id"))

	var attachment models.Attachment
	if err := database.DB.First(&attachment, attachmentID).Error; err != nil {
		utils.Error(c, 4004, "附件不存在")
		return
	}

	if attachment.Storage == models.StorageURL {
		c.Redirect(302, attachment.URL)
	} else {
		c.File(attachment.ObjectKey)
	}
}
