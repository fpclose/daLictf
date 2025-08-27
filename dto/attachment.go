// file: dto/attachment.go
package dto

type AddAttachmentURLReq struct {
	URL      string `json:"url" binding:"required,url"`
	FileName string `json:"file_name" binding:"required"`
}
